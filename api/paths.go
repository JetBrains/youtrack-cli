package api

const (
	// Admin / Global Settings
	pathAdminGlobalSettings              = "/api/admin/globalSettings"
	pathAdminGlobalSettingsNotifications = "/api/admin/globalSettings/notificationSettings"
	pathAdminFeatureFlag                 = "/api/admin/featureFlags/%s"
	pathGrazieConfig                     = "/api/grazie/config"
	pathConfig                           = "/api/config"
	pathAdminPluginUsages                = "/api/admin/pluginUsages"
	pathAdminPluginUsage                 = "/api/admin/pluginUsages/%s"

	// VCS / Integrations
	pathAdminVCSServers     = "/api/admin/integrations/vcsHostingServers"
	pathAdminVCSServer      = "/api/admin/integrations/vcsHostingServers/%s"
	pathAdminVCSServerProcs = "/api/admin/integrations/vcsHostingServers/%s/changesProcessors"
	pathAdminVCSProcessors  = "/api/admin/integrations/vcsHostingProcessors"
	pathAdminVCSProcessor   = "/api/admin/integrations/vcsHostingProcessors/%s"
	pathAdminMailboxes      = "/api/admin/integrations/mailbox/mailboxes"
	pathAdminMailbox        = "/api/admin/integrations/mailbox/mailboxes/%s"
	pathAdminMailboxRules   = "/api/admin/integrations/mailbox/rules"

	// Apps
	pathAdminApps = "/api/admin/apps"
	pathAdminApp  = "/api/admin/apps/%s"

	// Articles
	pathArticles           = "/api/articles"
	pathArticle            = "/api/articles/%s"
	pathArticleAttachments = "/api/articles/%s/attachments"
	pathArticleComments    = "/api/articles/%s/comments"
	pathArticleComment     = "/api/articles/%s/comments/%s"
	pathArticleTags        = "/api/articles/%s/tags"

	// Boards (Agiles) / Sprints
	pathBoards       = "/api/agiles"
	pathBoard        = "/api/agiles/%s"
	pathBoardSprints = "/api/agiles/%s/sprints"

	// Canvas
	pathPlanningCanvases    = "/api/planningCanvases"
	pathPlanningCanvas      = "/api/planningCanvases/%s"
	pathPlanningCanvasCards = "/api/planningCanvases/%s/cards"
	pathPlanningCanvasLinks = "/api/planningCanvases/%s/links"

	// Dashboards
	pathDashboards = "/api/dashboards"
	pathDashboard  = "/api/dashboards/%s"

	// Custom Fields / Bundles
	pathAdminCustomFields      = "/api/admin/customFieldSettings/customFields"
	pathAdminCustomField       = "/api/admin/customFieldSettings/customFields/%s"
	pathAdminCustomFieldBundle = "/api/admin/customFieldSettings/customFields/%s/fieldDefaults/bundle"
	pathAdminBundles           = "/api/admin/customFieldSettings/bundles/%s"
	pathAdminBundle            = "/api/admin/customFieldSettings/bundles/%s/%s"
	pathAdminBundleValues      = "/api/admin/customFieldSettings/bundles/%s/%s/values"
	pathAdminWorkItemTypes     = "/api/admin/timeTrackingSettings/workItemTypes"
	pathAdminWorkTimeSettings  = "/api/admin/timeTrackingSettings/workTimeSettings"

	// Groups
	pathGroups = "/api/groups"
	pathGroup  = "/api/groups/%s"

	// Issues
	pathIssues                = "/api/issues"
	pathIssue                 = "/api/issues/%s"
	pathIssueComments         = "/api/issues/%s/comments"
	pathIssueComment          = "/api/issues/%s/comments/%s"
	pathIssueCommentReactions = "/api/issues/%s/comments/%s/reactions"
	pathIssueCommentReaction  = "/api/issues/%s/comments/%s/reactions/%s"
	pathIssueLinks            = "/api/issues/%s/links"
	pathIssueLink             = "/api/issues/%s/links/%s"
	pathIssueLinkTypeIssues   = "/api/issues/%s/links/%s/issues"
	pathIssueTags             = "/api/issues/%s/tags"
	pathIssueAttachments      = "/api/issues/%s/attachments"
	pathIssueAttachment       = "/api/issues/%s/attachments/%s"
	pathIssueWorkItems        = "/api/issues/%s/timeTracking/workItems"
	pathIssueFields           = "/api/issues/%s/customFields"
	pathIssueField            = "/api/issues/%s/customFields/%s"
	pathIssueDrafts           = "/api/users/%s/drafts"
	pathIssueWatchers         = "/api/issues/%s/watchers"
	pathIssueVoters           = "/api/issues/%s/voters"
	pathIssueLinkTypes        = "/api/issueLinkTypes"
	pathIssuesGetterCount     = "/api/issuesGetter/count"
	pathCommands              = "/api/commands"
	pathCommandsAssist        = "/api/commands/assist"

	// Organizations
	pathAdminOrganizations = "/api/admin/organizations"
	pathAdminOrganization  = "/api/admin/organizations/%s"

	// Projects
	pathAdminProjects              = "/api/admin/projects"
	pathAdminProject               = "/api/admin/projects/%s"
	pathAdminProjectFields         = "/api/admin/projects/%s/fields"
	pathAdminProjectTeam           = "/api/admin/projects/%s/team"
	pathAdminProjectTeamOwnUsers   = "/api/admin/projects/%s/team/ownUsers"
	pathAdminProjectTeamGroups     = "/api/admin/projects/%s/team/groups"
	pathAdminProjectWorkflows      = "/api/admin/projects/%s/workflows"
	pathAdminProjectTimeTracking   = "/api/admin/projects/%s/timeTrackingSettings"
	pathAdminProjectTemplates      = "/api/admin/projectTemplates"
	pathAdminProjectHelpDesk       = "/api/admin/projects/%s/plugins/helpDeskSettings"
	pathAdminProjectHelpDeskAgents = "/api/admin/projects/%s/plugins/helpDeskSettings/agents"

	// Reports / Restored docs
	pathRemovedDocuments = "/api/removedDocuments"
	pathReport           = "/api/reports/%s"
	pathGanttCharts      = "/api/gantts"
	pathGanttChart       = "/api/gantts/%s"

	// Roles / Permissions
	pathRoles         = "/api/roles"
	pathRole          = "/api/roles/%s"
	pathPermissions   = "/api/permissions"
	pathAssignedRoles = "/api/assignedRoles"
	pathAssignedRole  = "/api/assignedRoles/%s"

	// Saved Searches
	pathSavedQueries = "/api/savedQueries"
	pathSavedQuery   = "/api/savedQueries/%s"

	// Tags
	pathTags = "/api/tags"
	pathTag  = "/api/tags/%s"

	// Users
	pathUsers                    = "/api/users"
	pathUser                     = "/api/users/%s"
	pathUserMe                   = "/api/users/me"
	pathUserProfileGeneral       = "/api/users/%s/profiles/general"
	pathUserProfileAppearance    = "/api/users/%s/profiles/appearance"
	pathUserProfileNotifications = "/api/users/%s/profiles/notifications"
	pathUserProfileGrazie        = "/api/users/%s/profiles/grazie"
	pathUserProfileArticles      = "/api/users/%s/profiles/articles"
	pathUserProfileIssuesList    = "/api/users/%s/profiles/issuesList"
	pathUserProfileTips          = "/api/users/%s/profiles/tips"
	pathUserProfileProperties    = "/api/users/%s/profiles/propertiesProfile/properties"
	pathUserDrafts               = "/api/users/%s/drafts"
	pathUserArticleDrafts        = "/api/users/%s/articleDrafts"
	pathUserPinnedProjects       = "/api/users/%s/pinnedProjects"

	// Workflows
	pathAdminWorkflows              = "/api/admin/workflows"
	pathAdminWorkflow               = "/api/admin/workflows/%s"
	pathAdminWorkflowRules          = "/api/admin/workflows/%s/rules"
	pathAdminWorkflowsRestoreStatus = "/api/admin/workflows/restoreStatus"
)
