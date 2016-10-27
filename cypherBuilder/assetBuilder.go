package cypherBuilder

import (
	"bytes"
	"html/template"
	"os"

	"github.com/kkirsche/trace2neo/trace2neolib"
)

func GetAssetTemplate() (*template.Template, error) {
	t := template.New("asset")

	t.Delims("[[", "]]")
	t, err := t.Parse("([[ .ShortName ]]:[[ .Label ]] {name:\"[[ .Name ]]\", IP:\"[[ .IPAddr ]]\"}),\n")
	if err != nil {
		return nil, err
	}
	return t, nil
}

func BuildAsset(t *template.Template, asset *trace2neolib.Asset) (string, error) {
	var assetBuf bytes.Buffer
	err := t.Execute(&assetBuf, asset)
	if err != nil {
		return "", err
	}
	builtAsset := assetBuf.String()
	return builtAsset, nil
}

func WriteAssetsToFile(a []string, fp string) error {
	f, err := os.OpenFile(fp, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("CREATE\n")

	for _, asset := range a {
		_, err = f.WriteString(asset)
		if err != nil {
			return err
		}
	}

	_, err = f.WriteString("(hop {name:\"Network hop\"});")

	return nil
}
