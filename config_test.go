package config

import (
	"flag"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestConfig_LoadFile(t *testing.T) {
	configData := []byte(`
	{
		"Start":10
	}
`)
	localConfigData := []byte(`
	{
		"Start":11
	}
`)
	name := writeToTempFile(configData, t)
	localName := writeToTempFile(localConfigData, t)

	c := config{}
	intOnly := &IntOnly{}
	require.NoError(t, c.LoadFile(name, localName, JSON, intOnly))
	require.Equal(t, 11, intOnly.Start)

	c.Generate()
	flag.VisitAll(func(fn *flag.Flag) {
		println(fn.Name, fn.DefValue, fn.Usage, fn.Value)
	})
	flag.Set("start", "12")
	require.Equal(t, 12, intOnly.Start)
	flag.Set("bool", "true")
	require.Equal(t, true, intOnly.Bool)
}

type IntOnly struct {
	Start int
	Bool  bool
}

func writeToTempFile(data []byte, t *testing.T) string {
	tempFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	_, err = tempFile.Write(data)
	require.NoError(t, err, "写入配置文件到临时文件")
	require.NoError(t, tempFile.Close())
	return tempFile.Name()
}

func TestConfig_LoadFileComplex(t *testing.T) {
	configData := []byte(`
	{
		"start":10,
		"end":{
			"start":10
		}
	}
`)
	localConfigData := []byte(`
	{
		"start":11,
		"end":{
			"start":11
		}
	}
`)
	name := writeToTempFile(configData, t)
	localName := writeToTempFile(localConfigData, t)

	c := config{}
	intOnly := &IntWithStruct{}
	require.NoError(t, c.LoadFile(name, localName, JSON, intOnly))
	require.Equal(t, 11, intOnly.End.Start)

	c.Generate()
	flag.VisitAll(func(fn *flag.Flag) {
		println(fn.Name, fn.DefValue, fn.Usage, fn.Value)
	})
	flag.Set("start", "12")
	require.Equal(t, 12, intOnly.Start)

	flag.Set("end.start", "12")
	require.Equal(t, 12, intOnly.End.Start)

	flag.Set("end.bool", "true")
	require.True(t, true, intOnly.End.Bool)

}

type IntWithStruct struct {
	Start int     `json:"start"`
	End   IntOnly `json:"end"`
}

func TestConfig_LoadFromServer(t *testing.T) {
	c := config{}
	intOnly := &IntWithStruct{}
	require.NoError(t, c.LoadFromServer("http://localhost:1234", "1", intOnly))
	spew.Dump(intOnly)
	c.Generate()
	flag.VisitAll(func(fn *flag.Flag) {
		println(fn.Name, fn.DefValue, fn.Usage, fn.Value)
	})
}
