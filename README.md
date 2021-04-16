# Communication link - ROS2 Node

## Generate a device key pair
```
openssl req -x509 -newkey rsa:2048 -keyout rsa_private.pem -nodes -out rsa_cert.pem -subj "/CN=unused"
```
The `rsa_private.pem` will stay on the device side and the public part `rsa_cert.pem` will be uploaded to the MQTT broker.

## Building

The application can be built as follows
```
source ../../install/setup.bash
go build
```

In addition to communication link, the video streaming node can be built as well from this repo.
The videonode listens a selected video stream based on the configuration file and streams it to rtsp server in cloud.
The video feed can be selected from cloud by sending mqtt messages to select a wanted camera configuration. Please see the image videonode/videostream.png. 
The videonode can be built as follows:
```
cd videonode
go build
```

## Running

```
./communication_link -device_id "<my-device-id>"
videonode/videonode -device_id "<my-device-id>"
```
*IMPORTANT*<br>
videonode has been removed and its functionality is now in depthai_ctrl ROS2 node.

