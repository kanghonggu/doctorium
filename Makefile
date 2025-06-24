build: proto
	go build -o build/doctoriumd ./cmd/doctoriumd

proto:
	buf generate proto/doctorium/filehash
