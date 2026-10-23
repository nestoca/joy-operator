package helm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type LoginParams struct {
	User     string
	Password []byte
	Registry string
}

func Login(ctx context.Context, params LoginParams) error {
	login := exec.CommandContext(ctx, "helm", "registry", "login", "-u", params.User, "--password-stdin", params.Registry)

	var buffer bytes.Buffer
	login.Stdout = &buffer
	login.Stderr = &buffer
	login.Stdin = bytes.NewReader(params.Password)

	if err := login.Run(); err != nil {
		return fmt.Errorf("%w: %q", err, &buffer)
	}

	return nil
}

func Logout(ctx context.Context, registry string) error {
	login := exec.CommandContext(ctx, "helm", "registry", "logout", registry)

	var buffer bytes.Buffer
	login.Stdout = &buffer
	login.Stderr = &buffer

	if err := login.Run(); err != nil {
		return fmt.Errorf("%w: %q", err, &buffer)
	}

	return nil
}

type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}
