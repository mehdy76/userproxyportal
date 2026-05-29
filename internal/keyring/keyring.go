package keyring

import (
	gokeyring "github.com/zalando/go-keyring"
)

const (
	service     = "userproxyportal"
	usernameKey = "username"
)

func Save(username, password string) error {
	if err := gokeyring.Set(service, usernameKey, username); err != nil {
		return err
	}
	return gokeyring.Set(service, username, password)
}

func Load() (username, password string, err error) {
	username, err = gokeyring.Get(service, usernameKey)
	if err != nil {
		return
	}
	password, err = gokeyring.Get(service, username)
	return
}

func Delete() {
	username, _ := gokeyring.Get(service, usernameKey)
	if username != "" {
		_ = gokeyring.Delete(service, username)
	}
	_ = gokeyring.Delete(service, usernameKey)
}
