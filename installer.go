/*
Copyright 2018 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package main

import "os"
import "io"
import "fmt"
import "bytes"
import "errors"
import "net/http"
import "io/ioutil"
import "crypto/md5"
import "encoding/hex"
import "encoding/json"

const (
	vUnstableURL = "http://api.vintagestory.at/latestunstable.txt"
	vStableURL   = "http://api.vintagestory.at/lateststable.txt"
	downloadURL  = "https://account.vintagestory.at/files/%v/%v"

	catalog1URL = "http://api.vintagestory.at/stable.json"
	catalog2URL = "http://api.vintagestory.at/unstable.json"
)

var versionFetchError = errors.New("Error parsing retrieved version information.")
var invalidSIDError = errors.New("Invalid or non-existent SID.")
var versionValidError = errors.New("Version validation failed.")
var md5ValidError = errors.New("MD5 validation failed.")

func GetLatestGameVersion(stable bool) (string, error) {
	url := vStableURL
	if !stable {
		url = vUnstableURL
	}

	client := new(http.Client)
	r, err := client.Get(url)
	if r != nil {
		defer r.Body.Close()
	}
	if err != nil {
		return ErrorVersion, err
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ErrorVersion, err
	}
	return string(bytes.TrimSpace(buf)), nil
}

func Download(ver string) error {
	ok, stable, file, srmd5 := ValidateVersion(ver)
	if !ok {
		return versionValidError
	}
	srsum := make([]byte, 16)
	l, err := hex.Decode(srsum, srmd5)
	if err != nil || l != 16 {
		return md5ValidError
	}

	s := "stable"
	if !stable {
		s = "unstable"
	}

	tr := &http.Transport{
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	r, err := client.Get(fmt.Sprintf(downloadURL, s, file))
	if r != nil {
		defer r.Body.Close()
	}
	if err != nil {
		return err
	}

	outfile, err := os.Create("./.dockvs-build/server.tar.gz")
	if err != nil {
		return err
	}
	defer outfile.Close()

	dlmd5 := md5.New()
	multiWriter := io.MultiWriter(dlmd5, outfile)
	_, err = io.Copy(multiWriter, r.Body)
	if err != nil {
		return err
	}
	dlsum := dlmd5.Sum(nil)
	if len(dlsum) != md5.Size || len(srsum) != md5.Size {
		return md5ValidError
	}
	for i := 0; i < md5.Size; i++ {
		if dlsum[i] != srsum[i] {
			return md5ValidError
		}
	}
	return nil
}

var ErrorVersion = "-1.-1.-1.-1"

// ValidateVersion checks if the game version could be found in the version catalog.
func ValidateVersion(v string) (ok bool, stable bool, file string, md5 []byte) {
	ok, file, md5 = validateVersion(catalog1URL, v)
	if ok {
		return ok, true, file, md5
	}
	ok, file, md5 = validateVersion(catalog2URL, v)
	return ok, false, file, md5
}

func validateVersion(url string, v string) (bool, string, []byte) {
	client := new(http.Client)
	r, err := client.Get(url)
	if r != nil {
		defer r.Body.Close()
	}
	if err != nil {
		return false, "", []byte{}
	}

	dec := json.NewDecoder(r.Body)
	catalog := make(map[string]map[string]vercatinfo)
	err = dec.Decode(&catalog)
	if err != nil {
		return false, "", []byte{}
	}

	vcat, found := catalog[v]
	if found {
		dat, ok := vcat["server"]
		if !ok {
			return false, "", []byte{}
		}
		return true, dat.File, []byte(dat.MD5)
	}
	return false, "", []byte{}
}

type vercatinfo struct {
	File string `json:"filename"`
	MD5  string `json:"md5"`
}
