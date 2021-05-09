package pacman

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/veecue/pacman-smartmirror/packet"
)

const fileGood = `%FILENAME%
acl-2.2.53-1-x86_64.pkg.tar.xz

%NAME%
acl

%BASE%
acl

%VERSION%
2.2.53-1

%DESC%
Access control list utilities, libraries and headers

%CSIZE%
135020

%ISIZE%
314368

%MD5SUM%
aaaea535e603f2b55cb320a42cc70397

%SHA256SUM%
27f4020c77a11992a75b5b99bc1c22797defcea6283b77eb2c311d77b3404443

%PGPSIG%
iQFCBAABCAAsFiEEAv0cepNOYUVFhJ8ZpiNAdEmOnO4FAlsoqCgOHGFyY2hAZXdvcm0uZGUACgkQpiNAdEmOnO4++ggApA70EbUN/n7Av4UL1fFQLGfMEqL5Fk6mdKbUc1IEHdmcR2C31MWlnUPk2zvbFj8qOpD76G8bq/JerfosPsapOXW+gHdJbI4fAHTYwYqlzymeUPEDkd6/8u/FVwVs0Hdm/0moy6n9c4uKth8XF8d//Ak7i7+klxvNrvoEQM1BsFGLREGbu1ioHf3UmBZLl+kZWtM2Yv0F3F7OPCpLpWCmBhkV3FEVlYDeby3tL3bgaa0NXCbL0uBKcOTAKm1TNNJyTGTh6X8NZUqJTLxApcvF9+7qKeKnN9gEMO9NKjnpkcn+rcweIGITbae1PpTztDhwY60Gtgl7Jjr7F4PwsPN1Pw==

%URL%
http://savannah.nongnu.org/projects/acl

%LICENSE%
LGPL

%ARCH%
x86_64

%BUILDDATE%
1529391128

%PACKAGER%
Christian Hesse <arch@eworm.de>

%REPLACES%
xfsacl

%CONFLICTS%
xfsacl

%PROVIDES%
xfsacl

%DEPENDS%
attr
`

const gccGood = `%FILENAME%
gcc-9.1.0-2-x86_64.pkg.tar.xz

%NAME%
gcc

%BASE%
gcc

%VERSION%
9.1.0-2

%DESC%
The GNU Compiler Collection - C and C++ frontends

%GROUPS%
base-devel

%CSIZE%
35571360

%ISIZE%
145802240

%MD5SUM%
52953cc05c0b6a112af45006a4f33f62

%SHA256SUM%
0f4325bd1d88faa05d8b1fa424c213c7270570c6dc61b9c28870c43d379134d6

%PGPSIG%
iQEzBAABCAAdFiEE82kWh9hnuBtRzgfZu+Q3cUhzKKkFAl0OmrEACgkQu+Q3cUhzKKlP4gf/RMDdvdFMzkxpvjHcHdBVR8WerjwOcfsBRV0JodZEk+0Ecqk1PBobAzgLOxYJPYG1xnvWr9IqCm0/4dwcoEJPbxiKzPGt4POC5DG0M1wQx92dEkCQrA/E6St1IsICbju0zjesS4i0IbraXtWh/CLcXeo4o0VVQrCw0lxycA21ce5NrNWMYHhtH9qSabyoPI3nx9o8v2WZvDxUUkz/V1Rc0u1HKfiTP+UrVuZ9QpCsaQXc6RmzS9oNv5yvGl4BoKZ3yGhmGlflmPcpjAWXQ6AJbKpkScGpjfne8ChtwDGCQm3qcay77ZJo2EjCtoL+Gm2rKEl91lsIyQeW4M+kogWNgw==

%URL%
https://gcc.gnu.org

%LICENSE%
GPL
LGPL
FDL
custom

%ARCH%
x86_64

%BUILDDATE%
1561233125

%PACKAGER%
Bartłomiej Piotrowski <bpiotrowski@archlinux.org>

%REPLACES%
gcc-multilib

%PROVIDES%
gcc-multilib

%DEPENDS%
gcc-libs=9.1.0-2
binutils>=2.28
libmpc

%OPTDEPENDS%
lib32-gcc-libs: for generating code for 32-bit ABI

%MAKEDEPENDS%
binutils
libmpc
gcc-ada
doxygen
lib32-glibc
lib32-gcc-libs
python
subversion

%CHECKDEPENDS%
dejagnu
inetutils`

func createTestTar() []byte {
	// Create and add some files to the archive.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	var files = []struct {
		Name, Body string
	}{
		{"acl-2.2.53-1/desc", fileGood},
		{"gcc-9.1.0-2/desc", gccGood},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}
	gw.Close()

	return buf.Bytes()
}

func TestDBParser(t *testing.T) {
	packets := make([]packet.Packet, 0)
	err := i.ParseDB(bytes.NewReader(createTestTar()), func(pkg packet.Packet) {
		packets = append(packets, pkg)
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(packets))
	assert.Equal(t, packets[0].Name(), "acl")
	assert.Equal(t, packets[0].Version(), "2.2.53-1")

	assert.Equal(t, packets[1].Name(), "gcc")
	assert.Equal(t, packets[1].Version(), "9.1.0-2")
}
