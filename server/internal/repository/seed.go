package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"campus-fellow/server/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// Seed is transactional and idempotent. Existing credentials are never overwritten.
func (r *Repository) Seed(ctx context.Context, cfg config.Config) error {
	return r.transaction(ctx, func(tx *sql.Tx) error {
		var admin int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE role='admin' ORDER BY id LIMIT 1").Scan(&admin)
		if err == sql.ErrNoRows {
			hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), 12)
			if err != nil {
				return err
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO users(username,password_hash,nickname,bio,role) VALUES (?,?,? ,?,'admin')`, cfg.AdminUsername, string(hash), "校园管理站", "维护社区秩序，也分享校园里的新鲜事。")
			if err != nil {
				return err
			}
			admin, err = result.LastInsertId()
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if !cfg.SeedDemo {
			return nil
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
			return err
		}
		if count > 1 {
			return nil
		}
		hash, err := bcrypt.GenerateFromPassword([]byte("Demo123!"), 10)
		if err != nil {
			return err
		}
		users := []struct{ username, nickname, bio string }{
			{"linzhi", "林知夏", "爱散步、摄影和一切有风的下午。"},
			{"zhouyu", "周屿", "计算机系大三，正在努力把生活过得松弛一点。"},
			{"chenmu", "陈木", "羽毛球新手，图书馆常驻选手。"},
		}
		ids := make([]int64, 0, 3)
		for _, u := range users {
			result, err := tx.ExecContext(ctx, "INSERT INTO users(username,password_hash,nickname,bio) VALUES (?,?,?,?)", u.username, string(hash), u.nickname, u.bio)
			if err != nil {
				return err
			}
			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		exchanges := []struct {
			author                      int64
			kind, title, content, image string
		}{
			{ids[0], "share", "在仙林，把秋天装进周末的六个小时", "从北大楼出发，沿梧桐道慢慢走。没有赶路的任务，只带了一台相机和一杯还温热的拿铁。下午在草坪上坐了很久，看社团排练，也听陌生人弹吉他。\n\n如果你也喜欢这样不设目的地的散步，欢迎在评论区交换路线。", "/demo/autumn-walk.svg"},
			{ids[1], "share", "一个人自习也能有节奏：我的图书馆续命歌单", "最近在准备考试，摸索出一套自己的自习节奏：前 50 分钟专注，休息时去接水，再用 10 分钟整理刚才没想明白的地方。\n\n歌单以纯音乐为主，不会抢走注意力。也想听听大家压箱底的自习好物。", "/demo/library.svg"},
			{admin, "news", "校园社团开放日：本周六，来和 40+ 社团碰面", "本周六 10:00—16:00，中心广场将举行秋季社团开放日。现场包括社团展演、互动体验和招新咨询。参与活动无需报名，请根据现场指引有序排队。", "/demo/clubs.svg"},
		}
		posts := make([]int64, 0, 3)
		for _, p := range exchanges {
			result, err := tx.ExecContext(ctx, "INSERT INTO exchange_posts(author_id,kind,title,content,image_url) VALUES (?,?,?,?,?)", p.author, p.kind, p.title, p.content, p.image)
			if err != nil {
				return err
			}
			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			posts = append(posts, id)
		}
		for _, c := range []struct {
			post, user int64
			text       string
		}{{posts[0], ids[2], "北大楼到天文台的路线也很好走，傍晚光特别好看。"}, {posts[1], ids[0], "推荐试试图书馆四楼靠窗的位置，下午很安静。"}} {
			if _, err := tx.ExecContext(ctx, "INSERT INTO comments(post_id,user_id,content) VALUES (?,?,?)", c.post, c.user, c.text); err != nil {
				return err
			}
		}
		for _, pair := range [][2]int64{{posts[0], ids[1]}, {posts[0], ids[2]}, {posts[1], ids[0]}} {
			if _, err := tx.ExecContext(ctx, "INSERT INTO likes(post_id,user_id) VALUES (?,?)", pair[0], pair[1]); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO favorites(post_id,user_id) VALUES (?,?)", posts[0], ids[1]); err != nil {
			return err
		}
		mates := []struct {
			author                    int64
			title                     string
			tags                      []string
			details                   string
			image                     *string
			location, duration, notes string
		}{
			{ids[0], "周末去紫金山看秋天，想找一位慢慢走路的同伴", []string{"户外", "周末", "摄影"}, "计划周六早上八点从学校出发，走蒋王庙线路，全程不赶时间。我带了相机，可以在山顶一起拍拍照。路线有一定体力要求，希望你有基础运动习惯。", str("/demo/hiking.svg"), "紫金山 · 蒋王庙入口", "周六 08:00—15:00", "雨天顺延。"},
			{ids[1], "长期英语口语搭子，每周两次图书馆练习", []string{"学习", "英语", "长期"}, "目标是摆脱哑巴英语，主题包括校园生活、技术学习和旅行。希望每周能在图书馆见面两次，每次四十分钟，互相纠正表达。水平不限，能坚持最重要。", nil, "仙林校区图书馆", "持续四周", "如果时间合适，也可以在线上进行。"},
			{ids[2], "今晚七点打羽毛球，缺一位双打搭档", []string{"运动", "今晚", "羽毛球"}, "已经订好 19:00—20:30 的场地，想找一位能一起打双打的同学。我是新手，主要是轻松运动，不追求输赢。", str("/demo/badminton.svg"), "东区体育馆 3 号场", "今晚 19:00—20:30", "自带球拍，场地费 AA。"},
		}
		for _, p := range mates {
			tags, err := json.Marshal(p.tags)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO mate_posts(author_id,title,tags,details,image_url,location,contact,duration,notes) VALUES (?,?,?,?,?,?,?,?,?)", p.author, p.title, string(tags), p.details, p.image, p.location, "同意申请后私聊交换", p.duration, p.notes); err != nil {
				return err
			}
		}
		return nil
	})
}
func str(s string) *string { return &s }
