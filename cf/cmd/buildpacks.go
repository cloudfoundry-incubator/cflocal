package cmd

import "github.com/sclevine/forge"

// TODO: merge the version URLs into a single JSON file with download URLs
var Buildpacks forge.SystemBuildpacks = []forge.Buildpack{
	{
		Name:       "staticfile_buildpack",
		URL:        "https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v{{.}}/staticfile-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/staticfile-buildpack",
	},
	{
		Name:       "java_buildpack",
		URL:        "https://github.com/cloudfoundry/java-buildpack/releases/download/v{{.}}/java-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/java-buildpack",
	},
	{
		Name:       "ruby_buildpack",
		URL:        "https://github.com/cloudfoundry/ruby-buildpack/releases/download/v{{.}}/ruby-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/ruby-buildpack",
	},
	{
		Name:       "nodejs_buildpack",
		URL:        "https://github.com/cloudfoundry/nodejs-buildpack/releases/download/v{{.}}/nodejs-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/nodejs-buildpack",
	},
	{
		Name:       "go_buildpack",
		URL:        "https://github.com/cloudfoundry/go-buildpack/releases/download/v{{.}}/go-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/go-buildpack",
	},
	{
		Name:       "python_buildpack",
		URL:        "https://github.com/cloudfoundry/python-buildpack/releases/download/v{{.}}/python-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/python-buildpack",
	},
	{
		Name:       "php_buildpack",
		URL:        "https://github.com/cloudfoundry/php-buildpack/releases/download/v{{.}}/php-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/php-buildpack",
	},
	{
		Name:       "dotnet_core_buildpack",
		URL:        "https://github.com/cloudfoundry/dotnet-core-buildpack/releases/download/v{{.}}/dotnet-core-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/dotnet-core-buildpack",
	},
	{
		Name:       "binary_buildpack",
		URL:        "https://github.com/cloudfoundry/binary-buildpack/releases/download/v{{.}}/binary-buildpack-v{{.}}.zip",
		VersionURL: "https://raw.githubusercontent.com/sclevine/cflocal-data/master/versions/binary-buildpack",
	},
}
