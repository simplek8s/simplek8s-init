package common

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	log "github.com/sirupsen/logrus"
)

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func IsStringInList(value string, list []string) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}

func IsDir(path string) error {
	if len(path) == 0 {
		err := fmt.Errorf("path %q is empty", path)
		return err
	}

	if stat, err := os.Stat(path); err != nil {
		return err
	} else if !stat.IsDir() {
		err := fmt.Errorf("path %q is not a directory", path)
		return err
	}
	return nil
}

func WriteTemplate(filename string, templates fs.FS, templateFilename string, data any) error {
	// Parse template
	tmpl, err := template.ParseFS(templates, templateFilename)
	if err != nil {
		log.WithFields(log.Fields{
			"templates": templates,
			"tmpl":      tmpl,
		}).Error(err)
		return err
	}
	log.Debug(tmpl)

	// Render template
	content := new(bytes.Buffer)
	if err := tmpl.Execute(content, data); err != nil {
		log.WithFields(log.Fields{
			"tmpl": tmpl,
			"data": data,
		}).Error(err)
		return err
	}
	log.Debug(content)

	// Create destination directory
	dirname := filepath.Dir(filename)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"dirname":  dirname,
		}).Error(err)
		return err
	}

	// Write file
	mode := fs.FileMode(0644)
	if err := os.WriteFile(filename, content.Bytes(), mode); err != nil {
		log.WithFields(log.Fields{
			"filename": filename,
			"content":  content.String(),
			"mode":     mode,
		}).Error(err)
		return err
	}

	return nil
}
