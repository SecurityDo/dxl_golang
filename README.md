# dxl_golang
DXL client in GoLang

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

This an OpenDXL client written in GoLang.

## Installation

> go get github.com/SecurityDo/dxl_golang

## Provisioning clients

DXL clients must be provisioned/configured prior to being able to connect to a DXL broker.

This will require authentication certificates and the address of the DXL broker.

To generate/obtain certicates, following instructions in the following link(from the DXL python documentation):

[External Certificate Authority (CA) Provisioning](https://opendxl.github.io/opendxl-client-python/pydoc/epoexternalcertissuance.html)

## Documentation

Refer to the OpenDXL python project's documentation for more detailed info regarding ePO configurations.

[Data Exchange Layer (DXL) Python SDK Documentation](https://opendxl.github.io/opendxl-client-python/pydoc/)

## Examples

Examples are found in the "examples" directory. The examples authenticates with the same config files obtained in the "Provisioning clients" section above.

Add/copy the contents of the "config/" directory to the "examples" directory.

Examples can be run via the golang test command:

> go test -v connect_test.go

(Include the -v flag to see output.)
