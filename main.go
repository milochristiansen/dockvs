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

// A simple tool for packaging and running Vintage Story game servers as Docker containers.
package main

import "os"
import "strings"
import "strconv"
import "io/ioutil"
import "os/exec"
import "fmt"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "build":
		err := os.Mkdir(".dockvs-build", 0777)
		if err != nil && !os.IsExist(err) {
			fmt.Println(err)
			os.Exit(1)
		}

		// Download the server files
		var ver string
		if len(os.Args) > 2 {
			ver = os.Args[2]
			switch ver {
			case "stable":
				ver, err = GetLatestGameVersion(true)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			case "unstable":
				ver, err = GetLatestGameVersion(false)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		} else {
			ver, err = GetLatestGameVersion(true)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
		err = Download(ver)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Create Dockerfile
		dfile, err := os.Create("./.dockvs-build/Dockerfile")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		_, err = fmt.Fprint(dfile, Dockerfile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		dfile.Close()

		// Run Docker
		cmd := exec.Command("docker", "build", "-t", "vs-"+strings.ToLower(ver), "./.dockvs-build")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	case "launch":
		if len(os.Args) < 3 {
			usage()
		}
		id := os.Args[2]
		// TODO: Sanitize ID
		err := os.Mkdir(id, 0777)
		if err != nil && !os.IsExist(err) {
			fmt.Println(err)
			os.Exit(1)
		}

		ver, port := "", "42420"
		sfile, err := ioutil.ReadFile("./" + id + "/.dockvs")
		if err == nil && !os.IsExist(err) {
			options := map[string]*string{
				"version": &ver,
				"port":    &port,
			}
			ParseINI(string(sfile), "\n", func(key, value string) {
				opt, ok := options[key]
				if ok {
					*opt = value
				}
			})
		}

		if len(os.Args) < 4 {
			// Use all saved settings.
		} else if len(os.Args) < 5 {
			ver = os.Args[3]

			err = ioutil.WriteFile("./"+id+"/.dockvs", []byte("\nversion="+ver+"\nport="+port+"\n"), 0666)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			ver, port = os.Args[3], os.Args[4]

			err = ioutil.WriteFile("./"+id+"/.dockvs", []byte("\nversion="+ver+"\nport="+port+"\n"), 0666)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		switch ver {
		case "":
			// if no version is provided or saved, try to use the latest stable. I hope you ran the build step first!
			fallthrough
		case "stable":
			ver, err = GetLatestGameVersion(true)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		case "unstable":
			ver, err = GetLatestGameVersion(false)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		default:
			// This is just a basic sanity check.
			ok, _, _, _ := ValidateVersion(ver)
			if !ok {
				fmt.Println("Invalid version number.")
				os.Exit(1)
			}
		}

		base, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cmd := exec.Command(
			"docker", "run", "-d", "-it", "--mount", "type=bind,source="+base+"/"+id+",target=/app/data",
			"--restart", "on-failure", "-p", port+":42420", "--name", id, "vs-"+strings.ToLower(ver))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		usage()
	}
}

var Dockerfile = `
FROM mono:latest
WORKDIR /app
ADD server.tar.gz bin
RUN mkdir data

EXPOSE 42420
CMD ["mono", "./bin/VintagestoryServer.exe", "--dataPath", "./data"]
`

func usage() {
	fmt.Println("Usage:")
	fmt.Println("dockvs build [stable|unstable|<version>]")
	fmt.Println("dockvs launch <id> [<version> [<port>]]")
	os.Exit(2)
}

// ParseINI is an extremely lazy INI parser.
// Malformed lines are silently skipped.
func ParseINI(input string, linedelim string, handler func(key, value string)) {
	lines := strings.Split(input, linedelim)
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		parts[0] = strings.TrimSpace(parts[0])
		parts[1] = strings.TrimSpace(parts[1])
		if un, err := strconv.Unquote(parts[1]); err == nil {
			parts[1] = un
		}
		handler(parts[0], parts[1])
	}
}
