package config

import (
	// "github.com/kylelemons/godebug/pretty"
	common "github.com/ncabatoff/process-exporter"
	. "gopkg.in/check.v1"
	"time"
)

func (s MySuite) TestConfigBasic(c *C) {
	yml := `
process_names:
  - exe: 
    - bash
  - exe: 
    - sh
  - exe: 
    - /bin/ksh
`
	cfg, err := GetConfig(yml, false)
	c.Assert(err, IsNil)
	c.Check(cfg.MatchNamers.matchers, HasLen, 3)

	bash := common.ProcAttributes{Name: "bash", Cmdline: []string{"/bin/bash"}}
	sh := common.ProcAttributes{Name: "sh", Cmdline: []string{"sh"}}
	ksh := common.ProcAttributes{Name: "ksh", Cmdline: []string{"/bin/ksh"}}

	found, name := cfg.MatchNamers.matchers[0].MatchAndName(bash)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "bash")
	found, name = cfg.MatchNamers.matchers[0].MatchAndName(sh)
	c.Check(found, Equals, false)
	found, name = cfg.MatchNamers.matchers[0].MatchAndName(ksh)
	c.Check(found, Equals, false)

	found, name = cfg.MatchNamers.matchers[1].MatchAndName(bash)
	c.Check(found, Equals, false)
	found, name = cfg.MatchNamers.matchers[1].MatchAndName(sh)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "sh")
	found, name = cfg.MatchNamers.matchers[1].MatchAndName(ksh)
	c.Check(found, Equals, false)

	found, name = cfg.MatchNamers.matchers[2].MatchAndName(bash)
	c.Check(found, Equals, false)
	found, name = cfg.MatchNamers.matchers[2].MatchAndName(sh)
	c.Check(found, Equals, false)
	found, name = cfg.MatchNamers.matchers[2].MatchAndName(ksh)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "ksh")
}

func (s MySuite) TestConfigTemplates(c *C) {
	yml := `
process_names:
  - exe: 
    - postmaster
    cmdline: 
    - "-D\\s+.+?(?P<Path>[^/]+)(?:$|\\s)"
    name: "{{.ExeBase}}:{{.Matches.Path}}"
  - exe: 
    - prometheus
    name: "{{.ExeFull}}:{{.PID}}"
  - comm:
    - cat
    name: "{{.StartTime}}"
`
	cfg, err := GetConfig(yml, false)
	c.Assert(err, IsNil)
	c.Check(cfg.MatchNamers.matchers, HasLen, 3)

	postgres := common.ProcAttributes{Name: "postmaster", Cmdline: []string{"/usr/bin/postmaster", "-D", "/data/pg"}}
	found, name := cfg.MatchNamers.matchers[0].MatchAndName(postgres)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "postmaster:pg")

	pm := common.ProcAttributes{
		Name:    "prometheus",
		Cmdline: []string{"/usr/local/bin/prometheus"},
		PID:     23,
	}
	found, name = cfg.MatchNamers.matchers[1].MatchAndName(pm)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "/usr/local/bin/prometheus:23")

	now := time.Now()
	cat := common.ProcAttributes{
		Name:      "cat",
		Cmdline:   []string{"/bin/cat"},
		StartTime: now,
	}
	found, name = cfg.MatchNamers.matchers[2].MatchAndName(cat)
	c.Check(found, Equals, true)
	c.Check(name, Equals, now.String())
}

func (s MySuite) TestExcludeConfig(c *C) {
	yml := `
process_names:
  - exe: 
    - "!postmaster"
    cmdline: 
    - "-D\\s+.+?(?P<Path>[^/]+)(?:$|\\s)"
    name: "{{.ExeBase}}:{{.Matches.Path}}"
  - exe: 
    - prometheus
    name: "{{.ExeFull}}:{{.PID}}"
  - comm:
    - "!cat"
    name: "{{.StartTime}}"
  - cmdline:
    - "!echo (?P<ExcludeArgs>.*)"
    name: "{{.ExeBase}}"
  - cmdline:
    - "!echo (?P<ExcludeArgs>.*)"
    - "cat (?P<IncludeArgs>.*)"
    name: "{{.Matches.ExcludeArgs}} {{.Matches.IncludeArgs}}"
`
	cfg, err := GetConfig(yml, false)
	c.Assert(err, IsNil)
	c.Check(cfg.MatchNamers.matchers, HasLen, 5)

	postgres := common.ProcAttributes{Name: "postmaster", Cmdline: []string{"/usr/bin/postmaster", "-D", "/data/pg"}}
	found, name := cfg.MatchNamers.matchers[0].MatchAndName(postgres)
	c.Check(found, Equals, false)
	c.Check(name, Equals, "")

	postgres2 := common.ProcAttributes{Name: "postmaster", Cmdline: []string{"/usr/bin/postmaster2", "-D", "/data/pg"}}
	found, name = cfg.MatchNamers.matchers[0].MatchAndName(postgres2)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "postmaster2:pg")

	pm := common.ProcAttributes{
		Name:    "prometheus",
		Cmdline: []string{"/usr/local/bin/prometheus"},
		PID:     23,
	}
	found, name = cfg.MatchNamers.matchers[1].MatchAndName(pm)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "/usr/local/bin/prometheus:23")

	now := time.Now()
	cat := common.ProcAttributes{
		Name:      "cat",
		Cmdline:   []string{"/bin/cat"},
		StartTime: now,
	}
	found, name = cfg.MatchNamers.matchers[2].MatchAndName(cat)
	c.Check(found, Equals, false)
	c.Check(name, Equals, "")

	echo := common.ProcAttributes{Name: "echo", Cmdline: []string{"/bin/echo", "$FILE"}}
	found, name = cfg.MatchNamers.matchers[3].MatchAndName(echo)
	c.Check(found, Equals, false)
	c.Check(name, Equals, "")

	cat = common.ProcAttributes{Name: "cat", Cmdline: []string{"/bin/cat", "$FILE_PATH"}}
	found, name = cfg.MatchNamers.matchers[3].MatchAndName(cat)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "cat")

	cat = common.ProcAttributes{Name: "cat", Cmdline: []string{"/bin/tail", "$FILE_PATH"}}
	found, name = cfg.MatchNamers.matchers[4].MatchAndName(cat)
	c.Check(found, Equals, false)
	c.Check(name, Equals, "")

	cat = common.ProcAttributes{Name: "cat", Cmdline: []string{"/bin/cat", "$FILE_PATH"}}
	found, name = cfg.MatchNamers.matchers[4].MatchAndName(cat)
	c.Check(found, Equals, true)
	c.Check(name, Equals, "<no value> $FILE_PATH")
}
