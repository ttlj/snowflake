KFlake
======

Example usage of the SnowFlake unique ID generator. By default it assumes running in a Kubernetes environment (expects environment variable MY_POD_NAME with StatefulSet pod-name).

```
Usage:
  -m value
    	comma separated MaskConfig values {time,worker,sequence} bits; default: 41,10,12
  -t string
    	worker-id type: {podid|podip|random}; default: podid
```

## Docker image

### Build

```
./linux64_build.sh

docker build . -t kflake
```


### Run docker image

```
export MY_POD_NAME="foobar-1"
docker run -p 3080:3080 kflake
```

With customized
```
export MY_POD_NAME="foobar-1"
docker run -p 3080:3080 kflake -m 42,10,11
```