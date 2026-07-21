# Unity UDP synchronization broker

This package provides an easy-to-use broker to connect 1..n devices communicating with a common protocol.
The protocol is based on [gzipped](http://www.gzip.org) [json](https://www.json.org/json-en.html),
bound to a specific [schema](vive-sync.schema.json).

The broker is written in [Go](https://www.golang.org) with several practical implications in mind:
* __Versatility__: JSON Schema must be easy to extend and new commands must be easily addable.
* __Portability__: Must be able to compile and run on any device w/o restrictions on OS or architecture.
* __Performance__: Performance leaks will interrupt study conduction (e.g. breaking immersion), so fluent performance is crucial.
* __Replayability__: Studies should be replayable for later validity testing or study revisions.
* __Throughput__: The transmitted data should be as little as possible to allow for multiple participating devices even on weak hardware.
* __Offline capability__: Studies should be conductible w/o internet connection.
* __Cloudlessness__: Studies should be conductible w/o cloud usage due to data privacy concerns.

## Installation
In any case, you need to clone the repo:
> `git clone git@github.com:hcig/unity-broker.git`

Then, you have 2 possibilities:

### Install the broker locally
1. Install go: https://go.dev/dl/
    1. We might provide a static windows/linux/mac binary in the future, thus no installation required.
2. Change into the folder and build the binary
> `cd unity-broker && go build`
3. Start the file
   1. Windows: `unity-broker.exe`
   2. Linux: `./unity-broker`

### Install the broker with docker/podman/containerd
If you have docker/podman/containerd available on your system, you don't need to install go in the first place.
The commands will be provided for docker only, but podman and containerd commands are similar:

> `docker build -t unity-broker -f Containerfile .`

This will build the broker using a dedicated go container and create another container with the tag __unity-broker__.
When the command finishes, you can start the broker with following command:

> `docker run -p 8397:8397 unity-broker`

The broker exposes UDP port 8397 (=HCIG) internally. You can change the locally exposed port by adjusting the `-p` flag.
If you e.g. want to locally expose the port `9999`, you can adjust the flag to `-p 9999:8397`.

## Extending the commands.
The schema and the overall broker architecture has been written in an extensible way.
If you want to add a command, you can add it to the [commands.go](commands.go) file as a function with the spec:
```go
func MyCommandHandler(com *Command, ch *CommandHandler) error {
  // Add your code here
  nil
}
```
and register it in the `RegisterCommands` function.

## Internal processes
The broker has a central PubSub broker, which every client can subscribe to.
By default, every client automatically is subscribed to the `basic` topic.
Besides, subscription-based notification, the server supports unicast and broadcast messages.

