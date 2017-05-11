package local

type Buildpack struct {
	Name       string
	URL        string
	VersionURL string
}

var Buildpacks BuildpackList = []Buildpack{
	{
		Name:       "staticfile_buildpack",
		URL:        "https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v{{.}}/staticfile-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/staticfile-buildpack/master/VERSION",
	},
	{
		Name:       "java_buildpack",
		URL:        "https://github.com/cloudfoundry/java-buildpack/releases/download/v{{.}}/java-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/java-buildpack",
	},
	{
		Name:       "ruby_buildpack",
		URL:        "https://github.com/cloudfoundry/ruby-buildpack/releases/download/v{{.}}/ruby-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/ruby-buildpack/master/VERSION",
	},
	{
		Name:       "nodejs_buildpack",
		URL:        "https://github.com/cloudfoundry/nodejs-buildpack/releases/download/v{{.}}/nodejs-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/nodejs-buildpack/master/VERSION",
	},
	{
		Name:       "go_buildpack",
		URL:        "https://github.com/cloudfoundry/go-buildpack/releases/download/v{{.}}/go-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/go-buildpack/master/VERSION",
	},
	{
		Name:       "python_buildpack",
		URL:        "https://github.com/cloudfoundry/python-buildpack/releases/download/v{{.}}/python-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/python-buildpack/master/VERSION",
	},
	{
		Name:       "php_buildpack",
		URL:        "https://github.com/cloudfoundry/php-buildpack/releases/download/v{{.}}/php-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/php-buildpack/master/VERSION",
	},
	{
		Name:       "dotnet_core_buildpack",
		URL:        "https://github.com/cloudfoundry/dotnet-core-buildpack/releases/download/v{{.}}/dotnet-core-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/dotnet-core-buildpack/master/VERSION",
	},
	{
		Name:       "binary_buildpack",
		URL:        "https://github.com/cloudfoundry/binary-buildpack/releases/download/v{{.}}/binary-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/cloudfoundry/binary-buildpack/master/VERSION",
	},
}
