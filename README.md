
DockVS Vintage Story Containerized Server Tools:
=======================================================================================================================

DockVS is a a simple tool for creating and launching containerized Vintage Story game servers with Docker. You still
need to know at least a few Docker basics to use this effectively, but the hardest parts are handled buy the tool.

Install:
-----------------------------------------------------------------------------------------------------------------------

To run DockVS you will need Docker and Go installed. 

* Make a new directory somewhere. This is where the tool and server game data will live.
* Run `go get github.com/milochristiansen/dockvs`
* Run `go build github.com/milochristiansen/dockvs` in the new directory created previously.

Now the tool is installed and ready to use.

Usage:
-----------------------------------------------------------------------------------------------------------------------

To create a server image run `./dockvs build <version>`, where `<version>` can be `stable`, `unstable`, or a specific
version number to build a Docker image for the given game version.

Once you have an image you can launch it with `./dockvs launch <id> <version> <port>`. `<id>` needs to be a unique
server ID, `<version>` is the game version you want to use (you can use `stable` or `unstable` as well), and `<port>`
is the port number you want to bind the server to. After the first time you launch this server you only need to provide
the same ID and it will load the version and port you used last. Additionally, when upgrading you only need to specify
the ID and new version.

Docker for Noobs:
-----------------------------------------------------------------------------------------------------------------------

Assuming you don't know a thing about Docker, here are some helpful commands. Note that `<id>` here is always the ID
you passed to the launch command for the server you want to talk to.

`docker attach <id>` will connect you to a running server and allow you to send commands to it. Use `ctrl+p ctrl+q` to
detach when done.

`docker logs <id>` will print the server output. You can add `--since 1h` to limit it to the output over the last hour
if you like (`docker logs --since 1h <id>`).

If the server hangs `docker restart <id>` will restart it. If the server hangs hard enough that it won't shutdown then
this command will forcibly kill it after 10 seconds.


Technical BS:
-----------------------------------------------------------------------------------------------------------------------

When building a container this tool downloads the required version to `./.dockvs-build/server.tar.gz`, then writes the
following Dockerfile to `./.dockvs-build/Dockerfile`:

	FROM mono:latest
	WORKDIR /app
	ADD server.tar.gz bin
	RUN mkdir data

	EXPOSE 42420
	CMD ["mono", "./bin/VintagestoryServer.exe", "--dataPath", "./data"]

To build the container it uses the following command:

	docker build -t vs-<version> ./.dockvs-build

When starting a container the command line looks like this:

	docker run -d -it --mount type=bind,source=<base>/<id>,target=/app/data --restart on-failure -p <port>:42420 --name <id> vs-<version>
