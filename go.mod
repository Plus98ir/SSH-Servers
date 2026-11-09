module sshservers

go 1.24

require (
	github.com/jchv/go-webview2 v0.0.0-20260205173254-56598839c808
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	golang.org/x/crypto v0.41.0
)

require (
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	golang.org/x/sys v0.35.0 // indirect
)

replace golang.org/x/crypto => github.com/golang/crypto v0.41.0

replace golang.org/x/sys => github.com/golang/sys v0.35.0

replace golang.org/x/net => github.com/golang/net v0.43.0

replace golang.org/x/term => github.com/golang/term v0.34.0

replace golang.org/x/text => github.com/golang/text v0.28.0
