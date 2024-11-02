repo_name:= asadlive84
image_name:=todo-backend
tag_name:=v1.1

build-push:
	@ docker build -t ${repo_name}/${image_name}:${tag} DockerFile
	@ docker push ${repo_name}/${image_name}:${tag}