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
	indexes := make(map[string]string)
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if info.Name() == "index.html" {
			dir := filepath.Dir(path)
			if dir == "." {
				dir = ""
			}
			indexes[dir] = path
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

handlers:{{ range $key, $path := . }}
- url: /{{$key}}
  static_files: public/{{$path}}
  upload: public/{{$path}}
{{ end }}
- url: /.*
  static_dir: public
`
