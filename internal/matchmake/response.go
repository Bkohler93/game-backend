package matchmake

type MatchResponse struct {
	UserOneId   string `redis:"user_one_id" mapstructure:"user_one_id"`
	UserOneName string `redis:"user_one_name" mapstructure:"user_one_name"`
	UserTwoId   string `redis:"user_two_id" mapstructure:"user_two_id"`
	UserTwoName string `redis:"user_two_name" mapstructure:"user_two_name"`
}
