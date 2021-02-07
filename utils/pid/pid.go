package pid

import (
"fmt"
"io/ioutil"
"os"
"strings"
)

type Release func()

// Record file
func Record(file string) (Release, error) {
    if len(file) == 0 {
        items := strings.Split(os.Args[0], "/")
        file = fmt.Sprintf("%s.pid", items[len(items)-1])
    }

    err := ioutil.WriteFile(file, []byte(fmt.Sprintf("%v", os.Getegid())), os.FileMode(0644))
    if err != nil {
        return nil, err
    }

    return func() {
        os.Remove(file)
    }, nil
}
