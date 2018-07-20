package types

type TerminalBody struct {
	Image string
}

type ResponseBody struct {
	Data  string
	Error string
}

const TerminalPod = `
{
	"apiVersion": "v1",
	"kind": "Pod",
	"metadata": {
		"name": "terminal-1",
		"namespace": "kfcoding-alpha",
		"labels": {
			"app": "terminal-1"
		}
	},
	"spec": {
		"containers": [
			{
				"name": "application",
				"command": [
					"/bin/sh",
					"-c",
					"--"
				],
				"args": [
					"while true; do sleep 30; done;"
				],
				"image": "ubuntu:latest"
			}
		]
	}
}
`
