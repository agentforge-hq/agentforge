//go:build ignore

// Build installable skill bundles using only Go's standard library and toolchain.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	forge "github.com/agentforge-hq/agentforge"
)

func main() {
	all := flag.Bool("all", false, "Build Linux, macOS and Windows bundles for amd64 and arm64")
	out := flag.String("out", "dist", "Output directory")
	flag.Parse()
	if err := build(*out, *all); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func build(out string, all bool) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return fmt.Errorf("run from the AgentForge source directory: %w", err)
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	targets := []string{runtime.GOOS + "/" + runtime.GOARCH}
	if all {
		targets = []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64"}
	}
	checksums := []string{}
	for _, target := range targets {
		parts := strings.Split(target, "/")
		goos, arch := parts[0], parts[1]
		staging, err := os.MkdirTemp(out, ".build-")
		if err != nil {
			return err
		}
		if err = buildTarget(root, out, staging, goos, arch, &checksums); err != nil {
			os.RemoveAll(staging)
			return err
		}
		os.RemoveAll(staging)
	}
	sort.Strings(checksums)
	return os.WriteFile(filepath.Join(out, "CHECKSUMS.txt"), []byte(strings.Join(checksums, "\n")+"\n"), 0644)
}
func buildTarget(root, out, staging, goos, arch string, checksums *[]string) error {
	name := "agentforge"
	if goos == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(staging, name)
	command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-s -w", "-o", binary, "./cmd/agentforge")
	command.Dir = root
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+arch, "GOTOOLCHAIN=local", "GOWORK=off")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build %s/%s: %w", goos, arch, err)
	}
	path := filepath.Join(out, "agentforge-v"+forge.Version+"-"+goos+"-"+arch+".zip")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	bundle := zip.NewWriter(file)
	add := func(source, destination string, mode os.FileMode) error {
		header := &zip.FileHeader{Name: "agentforge/" + destination, Method: zip.Deflate}
		header.SetMode(mode)
		writer, err := bundle.CreateHeader(header)
		if err != nil {
			return err
		}
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(writer, input)
		return err
	}
	for _, name := range []string{"SKILL.md", "README.md", "LICENSE", "NOTICE", "config.default.json"} {
		if err = add(filepath.Join(root, name), name, 0644); err != nil {
			bundle.Close()
			file.Close()
			return err
		}
	}
	err = filepath.WalkDir(filepath.Join(root, "references"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return add(path, filepath.ToSlash(relative), 0644)
	})
	if err != nil {
		bundle.Close()
		file.Close()
		return err
	}
	mode := os.FileMode(0755)
	if goos == "windows" {
		mode = 0644
	}
	if err = add(binary, "scripts/"+name, mode); err != nil {
		bundle.Close()
		file.Close()
		return err
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		bundle.Close()
		file.Close()
		return err
	}
	digest := sha256.Sum256(data)
	header := &zip.FileHeader{Name: "agentforge/build.json", Method: zip.Deflate}
	header.SetMode(0644)
	writer, err := bundle.CreateHeader(header)
	if err != nil {
		bundle.Close()
		file.Close()
		return err
	}
	err = json.NewEncoder(writer).Encode(map[string]string{"version": forge.Version, "os": goos, "arch": arch, "binary_sha256": hex.EncodeToString(digest[:]), "go_version": runtime.Version()})
	if err != nil {
		bundle.Close()
		file.Close()
		return err
	}
	if err = bundle.Close(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	data, err = os.ReadFile(path)
	if err != nil {
		return err
	}
	digest = sha256.Sum256(data)
	*checksums = append(*checksums, hex.EncodeToString(digest[:])+"  "+filepath.Base(path))
	fmt.Println(path)
	return nil
}
