package weeded

import (
	"github.com/dane-unltd/msglog"
)

type Aquire struct {
	f   *string
	UID uint64
	ret chan<- *msglog.Consumer
}

func manageFiles(aq chan *Aquire) {
	files := make(map[string]*File)
	fileNames := make(map[int]string)

	for {
		req := <-aq

		if req.f == nil {
			name, ok := files[req.conn.uid]
			if !ok {
				continue
			}
			buf, ok := buffers[name]
			if !ok {
				continue
			}
			buf.nUsers--
			buf.disconnect <- req.conn
			delete(files, req.conn.uid)

			fmt.Println("disconnecting:", req.conn.uid)
			if buf.nUsers == 0 {
				delete(buffers, name)
				err := ioutil.WriteFile(name, buf.b.Current, 0744)
				if err != nil {
					lg.Println(err)
				}
			}
			continue
		}

		buf, ok := buffers[*req.f]

		if !ok {
			var err error
			buf, err = NewBuffer(*req.f)
			if err != nil {
				lg.Println(err)
				continue
			}

			go buf.Run()
			buffers[*req.f] = buf
		}
		files[req.conn.uid] = *req.f

		buf.nUsers++
		buf.connect <- req.conn
		req.ret <- buf
	}
}
