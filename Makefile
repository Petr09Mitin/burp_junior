run:
	docker-compose build && docker-compose up

gen-ca:
	./utils/gen_ca.sh
