/*
Copyright 2020 Codelogia

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
)

// Server is the interface that wraps the Serve method.
type Server interface {
	Serve(addr, buildDir, token, appNamespace, appName, imageRegistry string) error
}

// New constructs a new Server.
func New() Server {
	return &server{}
}

type server struct{}

// Serve serves the build service for an app.
func (s *server) Serve(addr, buildDir, token, appNamespace, appName, imageRegistry string) error {
	stop := make(chan struct{}, 1)
	var once sync.Once

	router := http.NewServeMux()
	router.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		once.Do(func() {
			log.Println("receiving source...")

			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			tr := tar.NewReader(zr)
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				fileInfo := hdr.FileInfo()
				filePath := path.Join(buildDir, fileInfo.Name())
				if fileInfo.IsDir() {
					if err := os.MkdirAll(filePath, fileInfo.Mode()); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				} else {
					f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, fileInfo.Mode())
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if _, err := io.Copy(f, tr); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			}

			r.Body.Close()

			log.Println("building...")

			appNameWithRegistry := fmt.Sprintf("%s/%s/%s", imageRegistry, appNamespace, appName)

			cmd := exec.Command(
				"pack", "build", appNameWithRegistry,
				"--builder", "paketobuildpacks/builder:full",
				// "--builder", "heroku/buildpacks:18",
			)
			cmd.Dir = buildDir
			pipeReader, pipeWriter := io.Pipe()
			cmd.Stdout = io.MultiWriter(os.Stdout, pipeWriter)
			cmd.Stderr = io.MultiWriter(os.Stderr, pipeWriter)
			go write(pipeReader, w)
			if err := cmd.Run(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Println("pushing...")

			cmd = exec.Command(
				"docker", "push", appNameWithRegistry,
			)
			pipeReader, pipeWriter = io.Pipe()
			cmd.Stdout = io.MultiWriter(os.Stdout, pipeWriter)
			cmd.Stderr = io.MultiWriter(os.Stderr, pipeWriter)
			go write(pipeReader, w)
			if err := cmd.Run(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			stop <- struct{}{}
		})
	})

	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-stop
		httpServer.Shutdown(context.Background())
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func write(r io.ReadCloser, w io.Writer) {
	defer r.Close()
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			break
		}

		w.Write(buf[0:n])
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
