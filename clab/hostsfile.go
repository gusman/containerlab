package clab

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/srl-labs/containerlab/types"
	"github.com/srl-labs/containerlab/utils"
)

const (
	clabHostEntryPrefix  = "###### CLAB-%s-START ######"
	clabHostEntryPostfix = "###### CLAB-%s-END ######"
	clabHostsFilename    = "/etc/hosts"
)

func AppendHostsFileEntries(containers []types.GenericContainer, labname string) error {
	filename := clabHostsFilename
	if labname == "" {
		return fmt.Errorf("missing lab name")
	}
	if !utils.FileExists(filename) {
		err := utils.CreateFile(filename, "127.0.0.1\tlocalhost")
		if err != nil {
			return err
		}
	}
	// lets make sure to remove the entries of a non-properly destroyed lab in the hosts file
	err := DeleteEntriesFromHostsFile(labname)
	if err != nil {
		return err
	}
	data := generateHostsEntries(containers, labname)
	if len(data) == 0 {
		return nil
	}
	var f *os.File

	f, err = os.OpenFile(clabHostsFilename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// generateHostsEntries builds an /etc/hosts compliant text blob (as []byte]) for containers ipv4/6 address<->name pairs
func generateHostsEntries(containers []types.GenericContainer, labname string) []byte {

	entries := bytes.Buffer{}
	v6entries := bytes.Buffer{}

	fmt.Fprintf(&entries, clabHostEntryPrefix, labname)
	entries.WriteByte('\n')

	for _, cont := range containers {
		if len(cont.Names) == 0 {
			continue
		}
		if cont.NetworkSettings.Set {
			if cont.NetworkSettings.IPv4addr != "" {
				fmt.Fprintf(&entries, "%s\t%s\n", cont.NetworkSettings.IPv4addr, strings.TrimLeft(cont.Names[0], "/"))
			}
			if cont.NetworkSettings.IPv6addr != "" {
				fmt.Fprintf(&v6entries, "%s\t%s\n", cont.NetworkSettings.IPv6addr, strings.TrimLeft(cont.Names[0], "/"))
			}
		}
	}

	entries.Write(v6entries.Bytes())
	fmt.Fprintf(&entries, clabHostEntryPostfix, labname)
	entries.WriteByte('\n')
	return entries.Bytes()
}

func DeleteEntriesFromHostsFile(labname string) error {
	if labname == "" {
		return errors.New("missing containerlab name")
	}
	f, err := os.OpenFile(clabHostsFilename, os.O_RDWR, 0644) // skipcq: GSC-G302
	if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	skiplines := false
	output := bytes.Buffer{}
	prefix := fmt.Sprintf(clabHostEntryPrefix, labname)
	postfix := fmt.Sprintf(clabHostEntryPostfix, labname)
	for {
		line, err := reader.ReadString(byte('\n'))
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.TrimSpace(line) == postfix {
			skiplines = false
			continue
		} else if strings.TrimSpace(line) == prefix || skiplines {
			skiplines = true
			continue
		}
		output.WriteString(line)
	}
	if skiplines {
		// if skiplines is not false, we did not find the end
		// so we should not mess with /etc/hosts
		return fmt.Errorf("issue cleaning up %s file. Please do so manually", clabHostsFilename)
	}
	err = f.Truncate(0)
	if err != nil {
		return err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = f.Write(output.Bytes())
	return err
}
