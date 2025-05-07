all: bloop

bloop: src/*.go
	go build -o $@ ./src

data/million.txt: data/majestic_million.csv
	cat $< | tail -n 1000000 | tr , ' ' | awk '{print $$3}' > $@

data/majestic_million.csv: data
	@if [ ! -f "$@" ]; then \
		echo "Downloading majestic_million.csv..."; \
		wget -O "$@" https://downloads.majestic.com/majestic_million.csv; \
	fi

data:
	mkdir -p $@