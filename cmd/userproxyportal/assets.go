package main

import _ "embed"

//go:embed assets/userproxyportal.service
var embeddedService []byte

//go:embed assets/config.yaml.example
var embeddedConfig []byte
