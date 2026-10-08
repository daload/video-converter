package platform

import (
	"fmt"
	"os/exec"
)

func ShowError(message string) error {
	script := `on run arguments
	display alert "Conversor de Video" message (item 1 of arguments) as critical buttons {"Aceptar"} default button "Aceptar"
end run`
	output, err := exec.Command("/usr/bin/osascript", "-e", script, "--", message).CombinedOutput()
	if err != nil {
		return fmt.Errorf("No se pudo mostrar el error de inicio.\nDetalles técnicos: %w\n%s", err, output)
	}
	return nil
}
