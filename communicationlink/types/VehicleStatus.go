package types

import "unsafe"

/*
#cgo LDFLAGS: -L/opt/ros/foxy/lib -Wl,-rpath=/opt/ros/foxy/lib -lrcl -lrosidl_runtime_c -lrosidl_typesupport_cpp -lrosidl_typesupport_c -lstd_msgs__rosidl_generator_c -lpx4_msgs__rosidl_typesupport_c
#cgo CFLAGS: -I/opt/ros/foxy/include
#include <px4_msgs/msg/vehicle_status.h>
static inline const rosidl_message_type_support_t * ts_vstat(){
	const rosidl_message_type_support_t * ts =rosidl_typesupport_c__get_message_type_support_handle__px4_msgs__msg__VehicleStatus();
	return ts;
}
*/
import "C"

const (
	VehicleStatus_ARMING_STATE_INIT                       uint8 = 0
	VehicleStatus_ARMING_STATE_STANDBY                    uint8 = 1
	VehicleStatus_ARMING_STATE_ARMED                      uint8 = 2
	VehicleStatus_ARMING_STATE_STANDBY_ERROR              uint8 = 3
	VehicleStatus_ARMING_STATE_SHUTDOWN                   uint8 = 4
	VehicleStatus_ARMING_STATE_IN_AIR_RESTORE             uint8 = 5
	VehicleStatus_ARMING_STATE_MAX                        uint8 = 6
	VehicleStatus_FAILURE_NONE                            uint8 = 0
	VehicleStatus_FAILURE_ROLL                            uint8 = 1
	VehicleStatus_FAILURE_PITCH                           uint8 = 2
	VehicleStatus_FAILURE_ALT                             uint8 = 4
	VehicleStatus_FAILURE_EXT                             uint8 = 8
	VehicleStatus_FAILURE_ARM_ESC                         uint8 = 16
	VehicleStatus_HIL_STATE_OFF                           uint8 = 0
	VehicleStatus_HIL_STATE_ON                            uint8 = 1
	VehicleStatus_NAVIGATION_STATE_MANUAL                 uint8 = 0
	VehicleStatus_NAVIGATION_STATE_ALTCTL                 uint8 = 1
	VehicleStatus_NAVIGATION_STATE_POSCTL                 uint8 = 2
	VehicleStatus_NAVIGATION_STATE_AUTO_MISSION           uint8 = 3
	VehicleStatus_NAVIGATION_STATE_AUTO_LOITER            uint8 = 4
	VehicleStatus_NAVIGATION_STATE_AUTO_RTL               uint8 = 5
	VehicleStatus_NAVIGATION_STATE_AUTO_LANDENGFAIL       uint8 = 8
	VehicleStatus_NAVIGATION_STATE_AUTO_LANDGPSFAIL       uint8 = 9
	VehicleStatus_NAVIGATION_STATE_ACRO                   uint8 = 10
	VehicleStatus_NAVIGATION_STATE_UNUSED                 uint8 = 11
	VehicleStatus_NAVIGATION_STATE_DESCEND                uint8 = 12
	VehicleStatus_NAVIGATION_STATE_TERMINATION            uint8 = 13
	VehicleStatus_NAVIGATION_STATE_OFFBOARD               uint8 = 14
	VehicleStatus_NAVIGATION_STATE_STAB                   uint8 = 15
	VehicleStatus_NAVIGATION_STATE_RATTITUDE              uint8 = 16
	VehicleStatus_NAVIGATION_STATE_AUTO_TAKEOFF           uint8 = 17
	VehicleStatus_NAVIGATION_STATE_AUTO_LAND              uint8 = 18
	VehicleStatus_NAVIGATION_STATE_AUTO_FOLLOW_TARGET     uint8 = 19
	VehicleStatus_NAVIGATION_STATE_AUTO_PRECLAND          uint8 = 20
	VehicleStatus_NAVIGATION_STATE_ORBIT                  uint8 = 21
	VehicleStatus_NAVIGATION_STATE_MAX                    uint8 = 22
	VehicleStatus_RC_IN_MODE_DEFAULT                      uint8 = 0
	VehicleStatus_RC_IN_MODE_OFF                          uint8 = 1
	VehicleStatus_RC_IN_MODE_GENERATED                    uint8 = 2
	VehicleStatus_VEHICLE_TYPE_UNKNOWN                    uint8 = 0
	VehicleStatus_VEHICLE_TYPE_ROTARY_WING                uint8 = 1
	VehicleStatus_VEHICLE_TYPE_FIXED_WING                 uint8 = 2
	VehicleStatus_VEHICLE_TYPE_ROVER                      uint8 = 3
	VehicleStatus_VEHICLE_TYPE_AIRSHIP                    uint8 = 4
	VehicleStatus_ARM_DISARM_REASON_TRANSITION_TO_STANDBY uint8 = 0
	VehicleStatus_ARM_DISARM_REASON_RC_STICK              uint8 = 1
	VehicleStatus_ARM_DISARM_REASON_RC_SWITCH             uint8 = 2
	VehicleStatus_ARM_DISARM_REASON_COMMAND_INTERNAL      uint8 = 3
	VehicleStatus_ARM_DISARM_REASON_COMMAND_EXTERNAL      uint8 = 4
	VehicleStatus_ARM_DISARM_REASON_MISSION_START         uint8 = 5
	VehicleStatus_ARM_DISARM_REASON_SAFETY_BUTTON         uint8 = 6
	VehicleStatus_ARM_DISARM_REASON_AUTO_DISARM_LAND      uint8 = 7
	VehicleStatus_ARM_DISARM_REASON_AUTO_DISARM_PREFLIGHT uint8 = 8
	VehicleStatus_ARM_DISARM_REASON_KILL_SWITCH           uint8 = 9
	VehicleStatus_ARM_DISARM_REASON_LOCKDOWN              uint8 = 10
	VehicleStatus_ARM_DISARM_REASON_FAILURE_DETECTOR      uint8 = 11
	VehicleStatus_ARM_DISARM_REASON_SHUTDOWN              uint8 = 12
	VehicleStatus_ARM_DISARM_REASON_UNIT_TEST             uint8 = 13
)

type VehicleStatus struct {
	Timestamp                    uint64
	NavState                     uint8
	NavStateTimestamp            uint64
	ArmingState                  uint8
	HilState                     uint8
	Failsafe                     bool
        FailsafeTimestamp            uint64
	SystemType                   uint8
	SystemId                     uint8
	ComponentId                  uint8
	VehicleType                  uint8
	IsVtol                       bool
	IsVtolTailsitter             bool
	VtolFwPermanentStab          bool
	InTransitionMode             bool
	InTransitionToFw             bool
	RcSignalLost                 bool
	RcInputMode                  uint8
	DataLinkLost                 bool
	DataLinkLostCounter          uint8
	HighLatencyDataLinkLost      bool
	EngineFailure                bool
	MissionFailure               bool
	FailureDetectorStatus        uint8
	OnboardControlSensorsPresent uint32
	OnboardControlSensorsEnabled uint32
	OnboardControlSensorsHealth  uint32
	LatestArmingReason           uint8
	LatestDisarmingReason        uint8
        ArmedTime                    uint64
        TakeoffTime                  uint64
}

//TypeSupport ROS msg typesupport
func (t *VehicleStatus) TypeSupport() unsafe.Pointer {
	return unsafe.Pointer(C.ts_vstat())
}
