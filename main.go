// +build linux darwin

package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	tmpl, _ := template.New("app.yaml").Parse(appYamlTmpl)
	var indexes []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if info.Name() == "index.html" {
			websitePath := filepath.Dir(path)
			if websitePath == "." {
				return nil // add index.html manually
			}
			indexes = append(indexes, websitePath)
		}
		return nil
	})

	tmp, err := ioutil.TempDir("", "aestaticdeploy")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmp)

	f, err := os.OpenFile(filepath.Join(tmp, "app.yaml"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := tmpl.Execute(f, indexes); err != nil {
		log.Fatal(err)
	}
	f.Close()

	if err := exec.Command("cp", "-rf", ".", filepath.Join(tmp, "public")).Run(); err != nil {
		log.Fatal("cannot copy files to deployment tmp directory")
	}

	checkGcloud()
	cmd := exec.Command("gcloud", "app", "deploy")
	cmd.Dir = tmp
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func checkGcloud() {
	out, err := exec.Command("which", "gcloud").CombinedOutput()
	if err != nil {
		log.Fatal("Google Cloud SDK is not installed, see https://cloud.google.com/sdk/downloads.")
	}
	out, err = exec.Command("gcloud", "auth", "list").CombinedOutput()
	if err != nil || !bytes.ContainsAny(out, "ACTIVE") {
		log.Fatal("Google Cloud SDK has no authorized accounts, did you run `gcloud auth login`?")
	}
}

const appYamlTmpl = `runtime: python27
api_version: 1
threadsafe: true

handlers:
- url: /(.*\.ico)
  mime_type: image/x-icon
  static_files: public\1
  upload: public/(.*\.ico)

# index files
- url: /(.*)/
  static_files: public/\1/index.html
  upload: (.*)/index.html

- url: /img
  static_dir: public/img

- url: /css
  static_dir: public/css

- url: /att
  static_dir: public/att

# feed
- url: /index.xml
  static_files: public/index.xml
  upload: public/index.xml
  expiration: "15m"

# site root
- url: /
  static_files: public/index.html
  upload: public/index.html
  expiration: "15m"
`
