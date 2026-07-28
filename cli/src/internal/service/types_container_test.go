package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func unmarshalService(t *testing.T, y string) Service {
	t.Helper()
	var s Service
	require.NoError(t, yaml.Unmarshal([]byte(y), &s))
	return s
}

func TestService_Command_StringForm(t *testing.T) {
	s := unmarshalService(t, "host: local\nimage: node:20\ncommand: \"npm run dev\"\n")
	assert.Equal(t, "npm run dev", s.Command)
	assert.Nil(t, s.CommandArgs)
	assert.Equal(t, []string{"npm", "run", "dev"}, s.GetCommandArgs())
}

func TestService_Command_ArrayForm(t *testing.T) {
	s := unmarshalService(t, "host: local\nimage: postgres:16-alpine\ncommand:\n  - postgres\n  - -c\n  - max_connections=200\n")
	assert.Equal(t, []string{"postgres", "-c", "max_connections=200"}, s.CommandArgs)
	assert.Equal(t, "postgres -c max_connections=200", s.Command)
	assert.Equal(t, []string{"postgres", "-c", "max_connections=200"}, s.GetCommandArgs())
}

func TestService_Volumes(t *testing.T) {
	s := unmarshalService(t, "host: local\nimage: postgres:16-alpine\nvolumes:\n  - pgdata:/var/lib/postgresql/data\n  - ./init.sql:/docker-entrypoint-initdb.d/init.sql\n")
	require.Len(t, s.Volumes, 2)
	assert.Equal(t, "pgdata:/var/lib/postgresql/data", s.Volumes[0])
	assert.Equal(t, "./init.sql:/docker-entrypoint-initdb.d/init.sql", s.Volumes[1])
}

func TestService_PullPolicy(t *testing.T) {
	s := unmarshalService(t, "host: local\nimage: mcr.microsoft.com/azure-messaging/eventhubs-emulator:latest\npull_policy: missing\n")
	assert.Equal(t, "missing", s.PullPolicy)
}

func TestService_Command_InvalidType(t *testing.T) {
	var s Service
	// A mapping is not a valid command form.
	err := yaml.Unmarshal([]byte("host: local\nimage: x\ncommand:\n  foo: bar\n"), &s)
	assert.Error(t, err)
}

func TestService_NoContainerFields(t *testing.T) {
	// A plain service without the new fields still unmarshals cleanly.
	s := unmarshalService(t, "host: containerapp\nproject: ./api\nlanguage: python\n")
	assert.Empty(t, s.Volumes)
	assert.Empty(t, s.PullPolicy)
	assert.Empty(t, s.CommandArgs)
}

func TestService_RunsAsLocalProcess(t *testing.T) {
	// Top-level image = always a container (command is a container override).
	assert.False(t, (&Service{Image: "redis:7", Command: "redis-server --port 6380"}).RunsAsLocalProcess())
	// docker.image + explicit local command = process (docker.* is deploy-only).
	assert.True(t, (&Service{Docker: &DockerConfig{Image: "app/web"}, Command: "npm run dev"}).RunsAsLocalProcess())
	// docker.image + array command = process.
	assert.True(t, (&Service{Docker: &DockerConfig{Image: "app/web"}, CommandArgs: []string{"npm", "run", "dev"}}).RunsAsLocalProcess())
	// docker.image + type: process = process.
	assert.True(t, (&Service{Docker: &DockerConfig{Image: "app/web"}, Type: ServiceTypeProcess}).RunsAsLocalProcess())
	// docker.image, no local command/type = container (deploy default, unchanged).
	assert.False(t, (&Service{Docker: &DockerConfig{Image: "app/web"}}).RunsAsLocalProcess())
	// docker.path build service + command = process.
	assert.True(t, (&Service{Docker: &DockerConfig{Path: "./Dockerfile"}, Command: "npm run dev"}).RunsAsLocalProcess())
}
