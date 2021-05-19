// configuration logic
package main

import (
	"flag"
	"os"
    "io/ioutil"
    "encoding/json"
    "errors"
	"fmt"
)

var (
    ErrNoConfigFile = errors.New("Need a config file")
    ErrInConfigFile = errors.New("Error in config file")
)

type EngineConfiguration struct {
    Name     string             `json:"name"`
    Control  string             `json:"control"`
    Data     string             `json:"data"`
    Key      string             `json:"key"`
    Pubkey   string             `json:"pubkey"`
    Policies []PolicyEntryFile  `json:"policy"`
}

type PolicyEntryFile struct {
    DstSubnet   string `json:"dst"`
    SrcSubnet   string `json:"src"`
    Action      string `json:"action"`
    Endpoint    string `json:"endpoint"`
    ttl         int    `json:"ttl"`
}

type Configuration struct {
	Filename string
	content	 EngineConfiguration
}

func (c *Configuration) Init() (error) {

	c.parse()

	if c.Filename == "" {
		return ErrNoConfigFile
	}

	if err := c.ReadJSON(); err != nil {
        return ErrInConfigFile
    }

	return nil
}


// parse CLI arguments
func (c *Configuration) parse() () {

    version := flag.Bool("v", false, "Print version")
    configfile := flag.String ("c", "config.json", "Configuration file")
    flag.Parse()

    if (*version) {
        fmt.Println ("Wirelay - Version 0.1")
        os.Exit(0)
    }

    c.Filename = *configfile
}


func (c *Configuration) ReadJSON() (error) {
    var bytes []byte
    var err error

    if bytes, err = ioutil.ReadFile(c.Filename); err != nil {
        return err
    }

    if err := json.Unmarshal(bytes, &c.content); err != nil {
        return err
    }

    return nil
}

func (c *Configuration) SaveJSON() (error) {
    var bytes []byte
    var err error

    if bytes, err = json.Marshal(&c.content); err != nil {
        return err
    }

    if err = ioutil.WriteFile(c.Filename, bytes, 0644); err != nil {
        return err
    }

    return nil
}
