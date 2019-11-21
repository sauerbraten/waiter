# JSONfile

This is a Go package to easily parse commented JSON files into Go structs (or other types).

## Usage

Get the package:

	$ go get github.com/sauerbraten/jsonfile

Import the package:

	import (
		"github.com/sauerbraten/jsonfile"
	)

## Example

Write your commented JSON file (for example `config.json`) as a JSON object, like this (this is just an example of what a webserver configuration could look like):

	{
		// address to listen on
		"listen_address": "example.com",

		"listen_port": 8080, // port to listen on

		// directories containing static content
		"static_directories": ["css", "js", "html"],

		// ...
	}

In your Go code, have a struct ready to contain this configuration:

	type Config struct {
		ListenAddress     string   `json:"listen_address"`
		ListenPort        int      `json:"listen_port"`
		StaticDirectories []string `json:"static_directories"`
	}

Note: you don't have to provide the JSON key names if you use camelCase in your JSON file, too. Read more about it [here](http://golang.org/pkg/encoding/json/#Marshal).

**Make sure** the **fields** in your configuration struct are **exported**, i.e. uppercase, or else the `encoding/json` package will not be able to fill them.

Then in your Go code, do this:

	var config Config

	func init() {
		config = Config{}

		err := jsonfile.ParseFile("config.json", &config)
		if err != nil {
			log.Fatalln(err)
		}
	}

That's it, you can now easily access your configuration parameters in your code.

## Documentation

Proper documentation is at http://godoc.org/github.com/sauerbraten/jsonfile. There isn't much to say, really.

## License

This code is licensed under a BSD License:

Copyright (c) 2014-2015 Alexander Willing. All rights reserved.

- Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.