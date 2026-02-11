# Feature Specification: Automatic Meeting Recorder (Memofy v0.1)

**Feature Branch**: `001-auto-meeting-recorder`  
**Created**: February 12, 2026  
**Status**: Draft  
**Input**: User description: "macOS menu bar application that automatically records Teams/Zoom meetings via OBS using intelligent detection and stable state control"

## Clarifications

### Session 2026-02-12

- Q: What format should recording filenames use? → A: Timestamp + Application + Title format (e.g., `2026-02-12_1430_Zoom_Q1-Planning.mp4`)
- Q: How many consecutive detections are needed before starting recording? → A: 3 consecutive successful detections (6-9 seconds)
- Q: How should users configure detection rules for localized Teams/Zoom versions? → A: User configures rules during initial setup and can reconfigure anytime via Settings
- Q: How should the system notify users of critical errors? → A: Menu bar indicator + macOS system notification with actionable guidance
- Q: How many consecutive non-detections before stopping recording? → A: 6 consecutive non-detections (12-18 seconds)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automatic Meeting Detection and Recording (Priority: P1)

As a remote worker, I join Teams or Zoom meetings throughout my day and want them automatically recorded to my local machine without manual intervention, so I can focus on the meeting rather than managing recording controls.

**Why this priority**: This is the core value proposition - automatic, hands-free meeting capture. Without this, the application provides no advantage over manual OBS recording.

**Independent Test**: Can be fully tested by starting a Teams or Zoom meeting and verifying that recording begins automatically within 15 seconds, then stops automatically within 30 seconds of closing the meeting. Delivers immediate value by capturing meetings without user interaction.

**Acceptance Scenarios**:

1. **Given** recording backend is running and configured, **When** I start a Zoom meeting, **Then** the application detects the meeting and begins recording within 6-15 seconds
2. **Given** a recording is in progress, **When** I end the Zoom meeting, **Then** the recording stops within 12-25 seconds of the meeting ending
3. **Given** recording backend is running and configured, **When** I start a Microsoft Teams meeting, **Then** the application detects the meeting and begins recording within 6-15 seconds
4. **Given** a recording is in progress, **When** I end the Teams meeting, **Then** the recording stops within 12-25 seconds of the meeting ending
5. **Given** I have a short interruption (5-10 seconds) during a meeting, **When** the meeting continues, **Then** the application does not create multiple recording files (protected by 6-detection stop threshold)
6. **Given** no meeting is active, **When** I have Teams or Zoom running but not in a meeting, **Then** no recording is started (prevented by 3-detection start threshold)

---

### User Story 2 - Manual Recording Control Override (Priority: P2)

As a user, I sometimes want to manually start or stop recording regardless of automatic detection (e.g., for presentations, important discussions, or to exclude sensitive portions), so I have full control when needed.

**Why this priority**: Provides critical user control and handles edge cases where automatic detection may fail or user intent differs from detected state. Essential for user trust and practical usage.

**Independent Test**: Can be tested by manually triggering start/stop from the menu bar while in various states (meeting active, no meeting, auto mode). Delivers value by giving users confidence and control.

**Acceptance Scenarios**:

1. **Given** automatic mode is enabled but no meeting detected, **When** I click "Start Recording" from the menu bar, **Then** recording begins immediately regardless of detection state
2. **Given** a recording is in progress (auto or manual), **When** I click "Stop Recording" from the menu bar, **Then** recording stops immediately
3. **Given** manual recording mode is active, **When** I start or end a meeting, **Then** automatic detection is disabled and recording state does not change
4. **Given** manual mode is active, **When** I switch back to "Auto" mode, **Then** automatic detection resumes and adjusts recording state based on current meeting detection
5. **Given** I am in a meeting being recorded, **When** I click "Pause," **Then** the recording pauses and automatic detection is suspended until I resume

---

### User Story 3 - Status Monitoring and Configuration (Priority: P3)

As a user, I want to see the current recording status at a glance and configure detection sensitivity, so I can trust the system is working correctly and adapt it to my specific Teams/Zoom setup.

**Why this priority**: Enhances user confidence and handles localization/configuration variations, but the feature works without explicit monitoring if detection is reliable.

**Independent Test**: Can be tested by observing the menu bar icon states and accessing settings to modify detection hints. Delivers value by providing transparency and customization.

**Acceptance Scenarios**:

1. **Given** the application is running, **When** I click the menu bar icon, **Then** I see current status (IDLE/WAIT/REC/ERROR), active mode (Auto/Manual/Paused), and which application last triggered detection
2. **Given** the application is idle, **When** it detects a meeting and enters the waiting/debounce state, **Then** the menu bar icon updates to WAIT state
3. **Given** recording is in progress, **When** I view the menu bar dropdown, **Then** I see "REC" status and can access "Stop Recording" and "Open Recordings Folder"
4. **Given** an error occurs (OBS disconnected, permissions missing), **When** I check the menu bar, **Then** I see ERROR status with a brief description
5. **Given** I access Settings, **When** I modify Teams or Zoom window title hints, **Then** detection uses the new hints for future detection
6. **Given** I want to review what happened, **When** I click "Open Logs," **Then** the logs folder opens showing detection events and OBS actions

---

### Edge Cases

- What happens when recording backend is not running or not properly configured?
  - Application shows ERROR status in menu bar and sends macOS system notification with actionable guidance to configure recording backend
- What happens when required operating system permissions are not granted?
  - Application detects missing permissions, shows ERROR status in menu bar, and sends system notification with link to System Preferences
- What happens when I rapidly switch between meetings or join/leave multiple times?
  - Debounce state machine prevents recording fragmentation by requiring 6 consecutive non-detections (12-18 seconds) before stopping
- What happens when both Teams and Zoom are running simultaneously?
  - Detection prioritizes whichever application first shows meeting activity; tracks both independently
- What happens if recording backend connection is lost during recording?
  - Application attempts automatic reconnection and logs the event; shows ERROR status in menu bar and sends notification but does not falsely report recording state
- What happens if I manually delete or move the status file while the monitoring service is running?
  - Monitoring service recreates the status file on next update cycle
- What happens to the meeting title in the filename if it cannot be detected?
  - System uses generic placeholder (e.g., `2026-02-12_1430_Zoom_Meeting.mp4`) instead of specific title

## Requirements *(mandatory)*

### Functional Requirements

#### Meeting Detection

- **FR-001**: System MUST detect when user has joined a Zoom meeting
- **FR-002**: System MUST detect when user has joined a Microsoft Teams meeting
- **FR-003**: System MUST support configurable detection rules to handle different language versions and regional variations of meeting applications, with users configuring rules during initial setup and able to reconfigure anytime via Settings
- **FR-004**: System MUST check for active meetings at regular intervals (every 2-3 seconds)

#### State Management and Recording Control

- **FR-005**: System MUST require 3 consecutive successful meeting detections (6-9 seconds) before starting recording to prevent false triggers from brief application switches
- **FR-006**: System MUST require 6 consecutive non-detections (12-18 seconds) before stopping recording to prevent fragmentation from temporary disconnections
- **FR-007**: System MUST maintain connection to recording backend and recover from temporary disconnections
- **FR-008**: System MUST initiate recording only after confirming stable meeting state
- **FR-009**: System MUST stop recording only after confirming meeting has ended
- **FR-010**: System MUST verify current recording status before making state changes
- **FR-011**: System MUST support three operating modes: Auto (detection-based), Manual (user-controlled), and Paused (suspended)
- **FR-011A**: System MUST name recording files using format: `YYYY-MM-DD_HHMM_Application_Meeting-Title.mp4` (e.g., `2026-02-12_1430_Zoom_Q1-Planning.mp4`) where meeting title is included when detectable

#### Status Reporting and Command Interface

- **FR-012**: System MUST persist runtime status including: operating mode, detection state, actual recording state, which application triggered detection, state transition counters, last action, and errors
- **FR-013**: System MUST accept user commands including: start, stop, toggle, auto, pause, quit
- **FR-014**: System MUST update menu bar display to show one of four states: IDLE, WAIT (confirming meeting state), REC (recording), ERROR
- **FR-015**: Menu bar interface MUST display current mode (Auto/Manual/Paused) and which application last triggered detection (Teams/Zoom)
- **FR-016**: Menu bar interface MUST provide controls: Start Recording, Stop Recording, Auto Mode, Manual Mode, Pause
- **FR-017**: Menu bar interface MUST provide quick access to: Open Recordings Folder, Open Logs, Settings
- **FR-017A**: Settings interface MUST allow users to configure and update detection rules (window title hints, process patterns) for Teams and Zoom at any time

#### Permissions and Error Handling

- **FR-018**: System MUST verify required operating system permissions are granted before attempting to record
- **FR-018A**: System MUST notify users of critical errors through both menu bar ERROR indicator AND macOS system notifications with actionable guidance
- **FR-019**: System MUST verify recording backend is properly configured to capture screen content
- **FR-020**: System MUST log all detection events, state transitions, recording actions, and error recovery attempts
- **FR-021**: System MUST attempt automatic recovery when connection to recording backend is lost

#### Deployment and Lifecycle

- **FR-022**: Core monitoring service MUST start automatically when user logs in
- **FR-023**: Menu bar interface MUST be independently launchable and optional for users who prefer monitoring via status files
- **FR-024**: System MUST cleanly shut down and stop any active recording when receiving quit command

### Key Entities

- **Meeting Session**: Represents a detected meeting state including: meeting application (Teams/Zoom), detection timestamp, confirmation counter, stable state flag
- **Recording State**: Represents actual recording status including: recording active flag, start timestamp, backend confirmation status
- **Operating Mode**: Represents user control mode including: mode type (auto/manual/paused), last command received, mode transition timestamp
- **Detection Rule**: Represents configurable meeting detection criteria including: application name, matching patterns, display hints for different languages
- **Status Snapshot**: Represents point-in-time system state including: all detection flags, recording state, mode, timestamps, error conditions

### Dependencies and Assumptions

#### External Dependencies

- **Recording Backend**: System requires a compatible recording application to be installed and running
- **Operating System**: System targets macOS and requires operating system permissions for screen recording
- **Meeting Applications**: System supports Microsoft Teams and Zoom meeting applications

#### Assumptions

- Users have already configured their recording backend with appropriate settings (quality, storage location, etc.)
- Users grant necessary operating system permissions during initial setup
- Meeting applications use standard installation and window management patterns
- Users have sufficient disk space for recordings
- System has network connectivity for meeting applications to function

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: System correctly identifies meeting start within 15 seconds with 95% accuracy for standard Teams and Zoom configurations
- **SC-002**: System correctly identifies meeting end and stops recording within 30 seconds with 95% accuracy
- **SC-003**: Users experience zero recording file fragmentation for meetings longer than 2 minutes under normal conditions
- **SC-004**: Manual control commands (start/stop/pause) execute within 2 seconds of user interaction
- **SC-005**: System maintains stable OBS WebSocket connection with reconnection successful within 10 seconds for 99% of connection interruptions
- **SC-006**: Zero black screen recordings when required permissions are granted and recording backend is properly configured
- **SC-007**: Menu bar status updates reflect actual system state within seconds of state changes
- **SC-008**: 90% of users successfully configure detection hints for their localized Teams/Zoom versions within 5 minutes
