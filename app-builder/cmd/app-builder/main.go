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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codelogia/manor/app-builder/pkg/server"
)

const timeout = time.Minute * 10

func main() {
	addr := os.Getenv("ADDR")
	buildDir := os.Getenv("BUILD_DIR")
	token := os.Getenv("TOKEN")
	appNamespace := os.Getenv("APP_NAMESPACE")
	appName := os.Getenv("APP_NAME")
	imageRegistry := os.Getenv("IMAGE_REGISTRY")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("build dir: %s\n", buildDir)

	s := server.New()
	go func() {
		if err := s.Serve(
			addr,
			buildDir,
			token,
			appNamespace,
			appName,
			imageRegistry,
		); err != nil {
			os.RemoveAll(buildDir)
			log.Fatal(err)
		}
	}()

	select {
	case <-sigc:
		log.Println("terminating...")
	case <-ctx.Done():
		os.RemoveAll(buildDir)
		err := fmt.Errorf("build timed out")
		log.Fatal(err)
	}
}
