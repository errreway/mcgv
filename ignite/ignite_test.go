package ignite

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCorrectAddresses(t *testing.T) {
	cli, err := Start(ClientConfiguration{})

	assert.Nil(t, cli)
	assert.Error(t, err, "Addresses is empty!")

	cli, err = Start(ClientConfiguration{""})

	assert.Nil(t, cli)
	assert.Error(t, err, "Addresses is empty!")
}

func TestHandshake(t *testing.T) {
	cli, err := Start(ClientConfiguration{"localhost:10800"})

	assert.Nil(t, err)
	assert.NotNil(t, cli)
}
