package database

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const fileGood = `acl-2.2.53-1/someweird non chars%FILENAME%
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

func TestFilename(t *testing.T) {
	database, err := DbScratchFromGUnzippedReader(strings.NewReader(fileGood))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(database))
	assert.Equal(t, "acl-2.2.53-1-x86_64.pkg.tar.xz", database[0].Filename)
	assert.Equal(t, "acl", database[0].Name)
	assert.Equal(t, "acl", database[0].Base)
	assert.Equal(t, "2.2.53-1", database[0].Version)
	assert.Equal(t, "Access control list utilities, libraries and headers", database[0].Desc)
	assert.Equal(t, "135020", database[0].CSize)
	assert.Equal(t, "314368", database[0].ISize)
	assert.Equal(t, "aaaea535e603f2b55cb320a42cc70397", database[0].MD5Sum)
	assert.Equal(t, "iQFCBAABCAAsFiEEAv0cepNOYUVFhJ8ZpiNAdEmOnO4FAlsoqCgOHGFyY2hAZXdvcm0uZGUACgkQpiNAdEmOnO4++ggApA70EbUN/n7Av4UL1fFQLGfMEqL5Fk6mdKbUc1IEHdmcR2C31MWlnUPk2zvbFj8qOpD76G8bq/JerfosPsapOXW+gHdJbI4fAHTYwYqlzymeUPEDkd6/8u/FVwVs0Hdm/0moy6n9c4uKth8XF8d//Ak7i7+klxvNrvoEQM1BsFGLREGbu1ioHf3UmBZLl+kZWtM2Yv0F3F7OPCpLpWCmBhkV3FEVlYDeby3tL3bgaa0NXCbL0uBKcOTAKm1TNNJyTGTh6X8NZUqJTLxApcvF9+7qKeKnN9gEMO9NKjnpkcn+rcweIGITbae1PpTztDhwY60Gtgl7Jjr7F4PwsPN1Pw==", database[0].PGPSig)
	assert.Equal(t, "http://savannah.nongnu.org/projects/acl", database[0].URL)
	assert.Equal(t, "LGPL", database[0].License)
	assert.Equal(t, "x86_64", database[0].Arch)
	assert.Equal(t, "1529391128", database[0].BuildDate)
	assert.Equal(t, "Christian Hesse <arch@eworm.de>", database[0].Packager)
	assert.Equal(t, "xfsacl", database[0].Replaces)
	assert.Equal(t, "xfsacl", database[0].Conflicts)
	assert.Equal(t, "xfsacl", database[0].Provides)
	assert.Equal(t, "attr", database[0].Depends)

}
