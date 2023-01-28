package constant

var SystemAccounts = map[int64]*SystemAccount{}

type SystemAccount struct {
	Id        int64
	Name      string
	AvatarKey string
}

const (
	SYSTEM_ACCOUNT_ID_CONTACT_INVITE = 10000
)

const (
	SYSTEM_ACCOUNT_NAME_CONTACT_INVITE = "好友邀请"
)

const (
	SYSTEM_ACCOUNT_AVATAR_CONTACT_INVITE = "b11883ba-f3d7-4164-a593-700c177c37c8.jpeg"
)

func init() {
	SystemAccounts[SYSTEM_ACCOUNT_ID_CONTACT_INVITE] = &SystemAccount{
		Id:        SYSTEM_ACCOUNT_ID_CONTACT_INVITE,
		Name:      SYSTEM_ACCOUNT_NAME_CONTACT_INVITE,
		AvatarKey: SYSTEM_ACCOUNT_AVATAR_CONTACT_INVITE,
	}
}

func GetSystemAccount(id int64) (account *SystemAccount, ok bool) {
	account, ok = SystemAccounts[id]
	return
}
