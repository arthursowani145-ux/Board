package main

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
)

// Uploader describes how to push our log to a specific mirror.
// The Type field selects the mechanism; the rest is mechanism-specific.
type Uploader struct {
Type      string `json:"type"`
GistID    string `json:"gist_id,omitempty"`
MirrorURL string `json:"mirror_url,omitempty"`
}

func uploadersPath() (string, error) {
d, err := boardDir()
if err != nil {
return "", err
}
return filepath.Join(d, "uploaders.json"), nil
}

func loadUploaders() (map[string]Uploader, error) {
p, err := uploadersPath()
if err != nil {
return nil, err
}
data, err := os.ReadFile(p)
if os.IsNotExist(err) {
return map[string]Uploader{}, nil
}
if err != nil {
return nil, err
}
var out map[string]Uploader
if err := json.Unmarshal(data, &out); err != nil {
return nil, fmt.Errorf("uploaders: corrupt file: %w", err)
}
if out == nil {
out = map[string]Uploader{}
}
return out, nil
}

func saveUploaders(m map[string]Uploader) error {
p, err := uploadersPath()
if err != nil {
return err
}
data, err := json.MarshalIndent(m, "", "  ")
if err != nil {
return err
}
return os.WriteFile(p, data, filePerm)
}

func addUploader(name string, u Uploader) error {
if name == "" {
return fmt.Errorf("uploader: empty name")
}
m, err := loadUploaders()
if err != nil {
return err
}
m[name] = u
return saveUploaders(m)
}

func removeUploader(name string) error {
m, err := loadUploaders()
if err != nil {
return err
}
if _, ok := m[name]; !ok {
return fmt.Errorf("uploader: not found: %s", name)
}
delete(m, name)
return saveUploaders(m)
}
