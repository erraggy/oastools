package fileutil

import "os"

// OwnerReadWrite is the file permission mode for spec output files
// containing potentially sensitive API data (owner read/write only).
const OwnerReadWrite os.FileMode = 0o600

// ReadableByAll is the file permission mode for generated source code
// files intended to be read by build tools and other users.
const ReadableByAll os.FileMode = 0o644
