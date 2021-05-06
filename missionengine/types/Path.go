package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lnav_msgs__rosidl_typesupport_c -lnav_msgs__rosidl_generator_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <nav_msgs/msg/path.h>
static inline const rosidl_message_type_support_t * ts_path(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__nav_msgs__msg__Path();
    return ts;
}
*/
import "C"

//Path ROS Nav message struct
type Path struct {
	Header Header
	Poses  []PoseStamped
}

type CPath struct {
	Header Header
	Poses  CPoseData
}

type CPoseData struct {
	Data     uintptr
	Size     int
	Capacity int
}

//TypeSupport ROS msg typesupport
func (t *Path) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_path())
}

//GetData data getter
func (t *Path) GetData() unsafe.Pointer {
	result := CPath{
		Header: t.Header,
		Poses: CPoseData{
			Data:     uintptr(unsafe.Pointer(&t.Poses[0])),
			Size:     len(t.Poses),
			Capacity: len(t.Poses),
		},
	}

	return unsafe.Pointer(&result)
}

//Finish resource release
func (t *Path) Finish() {
	for _, pose := range t.Poses {
		C.free(unsafe.Pointer(pose.Header.FrameID.Data))
	}
}

//GeneratePath path generation
func GeneratePath() *Path {
	t := new(Path)
	t.Poses = make([]PoseStamped, 2)

	t.Poses[0].Header.FrameID.Data = C.CString("map")
	t.Poses[0].Header.FrameID.Size = 3
	t.Poses[0].Header.FrameID.Capacity = 4
	t.Poses[0].Header.Stamp = Time{100000, 1000000}
	t.Poses[0].Pose.Position.X = 1.0
	t.Poses[0].Pose.Position.Y = 2.0
	t.Poses[0].Pose.Position.Z = 3.0
	t.Poses[1].Header.FrameID.Data = C.CString("map")
	t.Poses[1].Header.FrameID.Size = 3
	t.Poses[1].Header.FrameID.Capacity = 4
	t.Poses[1].Header.Stamp = Time{100000, 1000000}
	t.Poses[1].Pose.Position.X = 1.0
	t.Poses[1].Pose.Position.Y = 2.0
	t.Poses[1].Pose.Position.Z = 3.0
	return t
}

// NewPath returns a valid path with given poses
func NewPath(waypoints []Point) *Path {
	p := Path{
		Header: Header{
			Stamp: Time{10000, 100000},
			FrameID: String{
				Data:     C.CString("map"),
				Size:     3,
				Capacity: 4,
			},
		},
		Poses: make([]PoseStamped, len(waypoints)),
	}
	for i, wp := range waypoints {
		p.Poses[i].Header.FrameID.Data = C.CString("map")
		p.Poses[i].Header.FrameID.Size = 3
		p.Poses[i].Header.FrameID.Capacity = 4
		p.Poses[i].Header.Stamp = Time{100000, 1000000}
		p.Poses[i].Pose.Position = wp
	}
	return &p
}
