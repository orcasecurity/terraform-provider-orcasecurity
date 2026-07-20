package api_client

// Automation action type IDs. These map an automation action to its integration
// backend and are consumed by the automation_v2 resource. They previously lived
// alongside the deprecated (removed) orcasecurity_automation resource.

// Alert lifecycle
const AutomationAlertDismissalID = 1
const AutomationAlertScoreChangeID = 28
const AutomationSnoozeID = 31

// Communication & Notification Actions
const AutomationSlackID = 2 // Covers both Slack v1 and Slack v2 integrations
const AutomationPagerDutyID = 3
const AutomationOpsgenieID = 4
const AutomationEmailID = 5
const AutomationMsTeamsID = 19

// SIEM & Security Tools
const AutomationSumoLogicID = 6
const AutomationAzureSentinelID = 7
const AutomationSplunkID = 8
const AutomationAWSSecurityHubID = 37
const AutomationChronicleID = 27
const AutomationSiemID = 21

// Ticketing & Project Management
const AutomationJiraID = 10
const AutomationJiraServerID = 23
const AutomationServiceNowIncidentsID = 22
const AutomationServiceNowSIIncidentsID = 24
const AutomationMondayID = 38
const AutomationLinearID = 40

// Cloud & Infrastructure
const AutomationGcpPubSubID = 13
const AutomationAwsSqsID = 33
const AutomationAwsSnsID = 34
const AutomationAwsSecurityLakeID = 25
const AutomationAzureDevopsID = 17

// Data & Analytics
const AutomationSnowflakeID = 26
const AutomationCoralogixID = 36
const AutomationDatadogID = 18
const AutomationCriblID = 29

// Automation & Orchestration
const AutomationWebhookID = 12
const AutomationTinesID = 30
const AutomationTorqID = 16

// Security & CDN
const AutomationCloudflareID = 32
const AutomationAkamaiID = 39
const AutomationPantherID = 41

// Remediation & Custom
const AutomationRemediationID = 20
const AutomationOpusID = 35

// Deprecated/Legacy
const AutomationGoogleSecurityOperationsSIEMID = 27 // Same as ChronicleID
