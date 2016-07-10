package envconf

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Var struct {
	Name    string
	Default string
	Desc    string
}

type Conf struct {
	env  map[string]string
	errs []*conferr
	vars []Var
}

func Parse() *Conf {
	env := make(map[string]string)
	for _, raw := range os.Environ() {
		pair := strings.SplitN(raw, "=", 2)
		env[pair[0]] = pair[1]
	}
	return &Conf{
		env: env,
	}
}

func (c *Conf) String(name, def, desc string) (string, bool) {
	v, ok := c.env[name]
	if !ok {
		return def, false
	}
	return v, true
}

func (c *Conf) Bool(name string, def bool, desc string) (bool, bool) {
	v, ok := c.env[name]
	if !ok {
		return def, false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b, true
	}
	return def, false
}

func (c *Conf) ReqString(name, desc string) string {
	v, ok := c.env[name]
	if !ok {
		c.errs = append(c.errs, &conferr{name, "required but not provided"})
	}
	return v
}

func (c *Conf) Int(name string, def int, desc string) (int, bool) {
	v, ok := c.env[name]
	if !ok {
		return def, false
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		// TODO: husio, there must be better way to pass erros (aggregate them?
		log.Printf("cannot parse %q: not int: %s", name, err)
		return def, false
	}
	return n, true
}

func (c *Conf) Finish() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-help", "-h", "help":
			// TODO: print help
			os.Exit(0)
		}
	}
	if c.errs == nil {
		return
	}
	for _, err := range c.errs {
		log.Printf("envconf: %s", err)
	}
	os.Exit(2)
}

type conferr struct {
	name string
	desc string
}

func (e *conferr) Error() string {
	return fmt.Sprintf("%s: %s", e.name, e.desc)
}
