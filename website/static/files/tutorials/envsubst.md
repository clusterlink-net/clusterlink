
Note that the provided YAML configuration files refer to environment variables
 (defined below) that should be set when running the tutorial. The values are
 replaced in the YAMLs using `envsubst` utility.

{{% expand summary="Installing `envsubst` on macOS" %}}
In case `envsubst` does not exist, you can install it with:

```sh
brew install gettext
brew link --force gettext
```

{{% /expand %}}
