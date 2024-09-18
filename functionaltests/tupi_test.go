package functionaltests

import (
	"net/http"
	"os/exec"
	"testing"
	"time"
)

func TestTupi(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	startServer()
	defer stopServer()

	var tests = []struct {
		name   string
		url    string
		status int
	}{
		{"get", "http://localhost:8080/something", 200},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", test.url, nil)
			c := http.Client{}
			resp, err := c.Do(r)

			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != test.status {
				t.Fatalf("bad status %d", resp.StatusCode)
			}

		})
	}

}

func startServer() {
	cmd := exec.Command("tupi", "-conf", "./../testdata/tupi-func.conf")
	if cmd.Err != nil {
		panic(cmd.Err.Error())
	}
	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}
	time.Sleep(time.Millisecond * 200)
}

func stopServer() {
	exec.Command("killall", "tupi")
}
