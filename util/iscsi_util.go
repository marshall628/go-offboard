package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jww "github.com/spf13/jwalterweatherman"
)

var tcp = "tcp"

func waitForPathToExist(devicePath string, maxRetries int,
	deviceTransport string) bool {
	// This makes unit testing a lot easier
	return waitForPathToExistInternal(devicePath, maxRetries, deviceTransport)
}

func waitForPathToExistInternal(devicePath string, maxRetries int,
	deviceTransport string) bool {
	for i := 0; i < maxRetries; i++ {
		var err error
		if deviceTransport == tcp {
			_, err = os.Stat(devicePath)
		} else {
			fpath, _ := filepath.Glob(devicePath)
			if fpath == nil {
				err = os.ErrNotExist
			}
		}
		if err == nil {
			return true
		}
		if err != nil && !os.IsNotExist(err) {
			return false
		}
		if i == maxRetries-1 {
			break
		}
		time.Sleep(time.Second)
	}
	return false
}

// extractTransportname: xtract the transport name
func extractTransportname(ifaceOutput string) (iscsiTransport string) {
	re := regexp.MustCompile(`iface.transport_name = (.*)\n`)

	regexoutput := re.FindStringSubmatch(ifaceOutput)
	if regexoutput != nil {
		iscsiTransport = regexoutput[1]
	} else {
		return ""
	}

	// While iface.transport_name is a required parameter, handle it being unspecified anyways
	if iscsiTransport == "<empty>" {
		iscsiTransport = "tcp"
	}
	return iscsiTransport
}

// read the json file to populate iscsiDiskMounter

type iscsiDiskMounter struct {
	iface  string
	fstype string
	portal string
	lun    string
	iqn    string
}

// ISCSIUtils structure for iscsi util
type ISCSIUtils struct{}

func (iscsiutil *ISCSIUtils) attachDisk(b *iscsiDiskMounter) error {
	var devicePath string
	var iscsiTransport string

	out, err := New().Command("iscsiadm", "-m", "iface", "-I", b.iface,
		"-o", "show").CombinedOutput()
	if err != nil {
		jww.ERROR.Printf("iscsi: could not read iface %s error: %s",
			b.iface, string(out))
		return err
	}

	// Extract the transport name
	iscsiTransport = extractTransportname(string(out))

	if iscsiTransport == "" {
		jww.ERROR.Printf("iscsi: could not find transport name in iface %s",
			b.iface)
		return fmt.Errorf("Could not parse iface file for %s", b.iface)
	} else if iscsiTransport == "tcp" {
		devicePath = strings.Join([]string{"/dev/disk/by-path/ip", b.portal,
			"iscsi", b.iqn, "lun", b.lun}, "-")
	} else {
		devicePath = strings.Join([]string{"/dev/disk/by-path/pci", "*", "ip",
			b.portal, "iscsi", b.iqn, "lun", b.lun}, "-")
	}
	exist := waitForPathToExist(devicePath, 1, iscsiTransport)
	if exist == false {
		// discover iscsi target
		out, err = New().Command("iscsiadm", "-m", "discovery", "-t",
			"sendtargets", "-p", b.portal, "-I", b.iface).CombinedOutput()
		if err != nil {
			jww.ERROR.Printf("iscsi: failed to sendtargets to portal %s error: %s",
				b.portal, string(out))
			return err
		}
		// login to iscsi target
		out, err = New().Command("iscsiadm", "-m", "node", "-p", b.portal,
			"-I", b.iface, "--login").CombinedOutput()
		if err != nil {
			jww.ERROR.Printf("iscsi: failed to attach disk:Error: %s (%v)",
				string(out), err)
			return err
		}
		exist = waitForPathToExist(devicePath, 10, iscsiTransport)
		if !exist {
			return errors.New("Could not attach disk: Timeout after 10s")
		}
	}
	return err
}
