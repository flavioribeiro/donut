# Developing using docker

To develop using docker, make sure you have `docker` installed. Run the following command to start simulations of SRT and RTMP streamings, and a bash session where you can run commands such as you'd in your local machine.

```bash
make run-docker-dev
```

Inside the container, you can start the donut server.

```bash
make run-server-inside-docker
```

You can access [http://localhost:8080/demo/](http://localhost:8080/demo/), using preferable the Chrome browser. You can connect to the simulated SRT and see donut working in practice.

You can work and change files locally, in your OS, and restart `CTRL+C + make run-server-inside-docker` the donut server in the container. It's fast because it avoids rebuilding all images. It'll offer a faster feedback cycle while developing.
