package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	data := []byte(`
feeds:
 first:
  title: "blah 1"
  sources:
   - name: nnn1
     url: http://aa.com/u1
   - name: nnn2
     url: http://aa.com/u2

 second:
  title: "blah 2"
  description: "some 2"
  sources:
   - name: mmm1
     url: https://bbb.com/u1

update: 600
`)

	assert.Nil(t, ioutil.WriteFile("/tmp/fm.yml", data, 0777), "failed write yml") // nolint

	r, err := loadConfig("/tmp/fm.yml")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(r.Feeds), "2 sets")
	assert.Equal(t, 2, len(r.Feeds["first"].Sources), "2 feeds in first")
	assert.Equal(t, 1, len(r.Feeds["second"].Sources), "1 feed in second")
	assert.Equal(t, "https://bbb.com/u1", r.Feeds["second"].Sources[0].URL)
}

func TestLoadConfigNotFoundFile(t *testing.T) {
	r, err := loadConfig("/tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "open /tmp/29e28b3c-e1a4-4269-a10b-3e9a89a08d45.txt: no such file or directory")
}

func TestLoadConfigInvalidYaml(t *testing.T) {
	assert.Nil(t, ioutil.WriteFile("/tmp/fm.txt", []byte(`Not Yaml`), 0777), "failed write yml") // nolint

	r, err := loadConfig("/tmp/fm.txt")

	assert.Nil(t, r)
	assert.EqualError(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `Not Yaml` into proc.Conf")
}
