package k8s

import (
	"embed"
	"fmt"

	"github.com/jlsalvador/simplek8s/internal/pkg/common"
)

//go:embed templates/*
var templates embed.FS

type KubeadmConf struct {
	// Example, `caravel.k8s.example.com`
	Hostname string
	// Could be `control-plane` or `worker`
	Role string
	// Could be `init` or `join`
	Verb string
	// Example, `k8s.example.com:6443`
	Endpoint string
	// Example, `aabbcc.defghijklmnopqrs`
	Token string
	// Example, `sha256:1b50bf4fa1424fbb023ed198dc1e9b1aa5bf61487f818715a4388595b4fcbdb1`
	TokenCaCertHash string
	// Example, `a18d09de1200842efdbd59a94f949c3fb43594a4fd6454f6d89818b52f0d4048`
	CertKey string
}

func getKubeadmConf(kubeadmConf KubeadmConf) (string, error) {
	var result string
	var err error

	if result, err = common.RenderTemplate(
		templates,
		"templates/kubeadm.conf.go.tmpl",
		kubeadmConf,
	); err != nil {
		return "", err
	}
	return result, nil
}

func RenderKubeadmConf() error {
	kubeadmConf := KubeadmConf{
		Hostname:        "levante",
		Role:            "worker",
		Verb:            "join",
		Endpoint:        "control-plane:6443",
		Token:           "pow8br.kpzjp557gn368ce6",
		TokenCaCertHash: "sha256:1a281c2287bd20b26a862bc4f3241a25afa571bbbd4bc6215f4ea0b5ef0b3fe9",
		CertKey:         "",
	}

	if output, err := getKubeadmConf(kubeadmConf); err != nil {
		panic(err)
	} else {
		fmt.Println(output)
	}
	return nil
}
