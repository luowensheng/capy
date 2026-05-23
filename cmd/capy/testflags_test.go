package main

import "flag"

// -update regenerates golden files in place when set.
var update = flag.Bool("update", false, "regenerate golden expected files")
