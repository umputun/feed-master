#!/usr/bin/env bash
set -eru -o pipefail
# exit on error
# exit on uninitialized variables
# enter restricted shell https://www.gnu.org/s/bash/manual/html_node/The-Restricted-Shell.html

URL='https://redirector.gvt1.com/edgedl/go/go1.9.2.src.tar.gz'
curl -L --silent "$URL" -o "go.tar.gz"
tar -zxf "go.tar.gz"

rm -rf "html/template/*"
cp "go/LICENSE" "./"
cp "go/PATENTS" "./"
cp "go/VERSION" "./"
cp -a "go/src/html/template/" "html/template/"
cp "export.go.tpl" "html/template/export.go"

rm -f "go.tar.gz"
rm -rf "./go/"
