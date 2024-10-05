tupi-cgi is a plugin for Tupi to allow the execution of cgi scripts

Install
=======

To install tupi-cgi first clone the code:

```sh
$ git clone https://github.com/jucacrispim/tupi-cgi
```

And then build the code:

```sh
$ cd tupi-cgi
$ make build
```

This will create the binary: ``./build/cgi_plugin.so``.

Usage
=====

To use the plugin with tupi, in  your config file put:

```toml
...
ServePlugin = "/path/to/cgi_plugin.so"
ServePluginConf = {
    "CGI_DIR" = "/path/to/somewhere"
}
...
```
