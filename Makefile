push:
	git add . && git commit -m "$(m)" && git push
docker:
	#Pass version using v variable
	sudo docker build  --platform linux/amd64 -t dafraer/messenger:$(v)-amd64 .
	docker push dafraer/messenger:$(v)-amd64
	sudo docker build  --platform linux/arm64 -t dafraer/messenger:$(v)-arm64 .
	docker push dafraer/messenger:$(v)-arm64
