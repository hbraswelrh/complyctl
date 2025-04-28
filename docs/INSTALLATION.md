# Installation

### Prerequisites

- **Go** version 1.20 or higher
- **Make** (optional, for using the `Makefile` if included)
- **pandoc** (optional, for generating man pages using the `make man`)

### Clone the repository

```bash
git clone https://github.com/complytime/complytime.git
cd complytime
```

### Build Instructions
To compile complytime and openscap-plugin:

```bash
make build
```

The binaries can be found in the `bin/` directory in the local repo. Add it to your PATH and you are all set!
