all: bloop

bloop: src/*.go src/data/million.txt
	go build -o $@ ./src

src/data/million.txt: src/data/majestic_million.csv
	cat $< | tail -n 1000000 | tr , ' ' | awk '{print $$3}' > $@

src/data/majestic_million.csv: src/data
	@if [ ! -f "$@" ]; then \
		echo "Downloading majestic_million.csv..."; \
		wget -O "$@" https://downloads.majestic.com/majestic_million.csv; \
	fi

src/data:
	mkdir -p $@