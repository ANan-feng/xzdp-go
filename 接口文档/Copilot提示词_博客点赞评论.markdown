# 博客点赞评论模块 - Copilot 实现提示词

请按照以下设计方案，为小众点评（xzdp-go）项目实现博客点赞评论模块。项目使用 Go + Gin + GORM + Redis（go-redis/v9），请严格遵循现有的代码风格和分层结构（controller → service → dao → model/utils）。

---

## 一、数据库表设计

### 1. blog 表

```sql
ALTER TABLE blog ADD COLUMN liked_count int unsigned DEFAULT 0 COMMENT '点赞数（冗余计数）';
ALTER TABLE blog ADD COLUMN comment_count int unsigned DEFAULT 0 COMMENT '评论数（冗余计数）';
-- 如果原表有 liked 和 comments 字段，先删除再添加上述字段
```

### 2. blog_like 表

```sql
CREATE TABLE blog_like (
  id bigint unsigned NOT NULL,
  user_id bigint unsigned NOT NULL,
  blog_id bigint unsigned NOT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_blog (user_id, blog_id) COMMENT '一人一点赞'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
```

### 3. blog_comment 表

```sql
CREATE TABLE blog_comment (
  id bigint unsigned NOT NULL,
  user_id bigint unsigned NOT NULL COMMENT '评论者id',
  blog_id bigint unsigned NOT NULL COMMENT '博客id',
  parent_id bigint unsigned DEFAULT 0 COMMENT '0=一级评论，否则为父评论id',
  reply_user_id bigint unsigned DEFAULT 0 COMMENT '回复的目标用户id（0=不是回复）',
  content varchar(500) NOT NULL COMMENT '评论内容',
  liked_count int unsigned DEFAULT 0 COMMENT '点赞数（冗余计数）',
  status tinyint unsigned DEFAULT 0 COMMENT '0正常 1举报 2禁看/删除',
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  INDEX idx_blog_parent (blog_id, parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
```

### 4. comment_like 表

```sql
CREATE TABLE comment_like (
  id bigint unsigned NOT NULL,
  user_id bigint unsigned NOT NULL,
  comment_id bigint unsigned NOT NULL,
  create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_comment (user_id, comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
```

---

## 二、Model 层

请创建以下 GORM Model（参考项目中已有的 model 结构）：

- `model/blog_like.go` → BlogLike 结构体
- `model/blog_comment.go` → BlogComment 结构体
- `model/comment_like.go` → CommentLike 结构体

注意：Blog 模型需要添加 `LikedCount` 和 `CommentCount` 字段，以及一个非数据库字段 `IsLike bool` 用于返回给前端。

---

## 三、Redis Key 设计

| Key 模式 | 类型 | 用途 | 说明 |
|----------|------|------|------|
| `blog:like:{blogId}` | SortedSet | 博客点赞 | score=时间戳, member=userId |
| `comment:like:{commentId}` | Set | 评论点赞 | member=userId |
| `blog:detail:{blogId}` | Hash | 博客详情缓存 | field为blog各字段 |
| `blog:comments:{blogId}:hot` | List | 热门一级评论ID列表 | 只缓存第1页10条 |
| `blog:comments:{blogId}:recent` | List | 最新一级评论ID列表 | 只缓存第1页10条 |
| `comment:detail:{commentId}` | Hash | 单条评论详情缓存 | field为comment各字段 |

请将 Redis Key 常量定义在 utils 包中，参考项目中已有的 key 命名风格。

---

## 四、DAO 层

创建 `dao/blog_dao.go` 和 `dao/comment_dao.go`，包含以下方法：

### BlogDAO

```go
// 博客点赞相关
func (d *BlogDAO) CreateBlogLike(ctx context.Context, userId, blogId int64) error
func (d *BlogDAO) DeleteBlogLike(ctx context.Context, userId, blogId int64) error
func (d *BlogDAO) CheckBlogLiked(ctx context.Context, userId, blogId int64) (bool, error)
func (d *BlogDAO) IncrBlogLikedCount(ctx context.Context, blogId int64) error   // liked_count + 1
func (d *BlogDAO) DecrBlogLikedCount(ctx context.Context, blogId int64) error   // liked_count - 1
func (d *BlogDAO) IncrBlogCommentCount(ctx context.Context, blogId int64) error // comment_count + 1
func (d *BlogDAO) DecrBlogCommentCount(ctx context.Context, blogId, count int64) error // comment_count - count

// 博客查询相关
func (d *BlogDAO) GetBlogById(ctx context.Context, blogId int64) (*model.Blog, error)
func (d *BlogDAO) GetBlogsByShopId(ctx context.Context, shopId int64, page, pageSize int) ([]*model.Blog, error)
```

### CommentDAO

```go
// 评论CRUD
func (d *CommentDAO) CreateComment(ctx context.Context, comment *model.BlogComment) error
func (d *CommentDAO) DeleteCommentLogical(ctx context.Context, commentId int64) error           // UPDATE status=2
func (d *CommentDAO) DeleteSubCommentsLogical(ctx context.Context, parentId int64) error         // 子评论也标记删除
func (d *CommentDAO) GetCommentById(ctx context.Context, commentId int64) (*model.BlogComment, error)
func (d *CommentDAO) GetHotComments(ctx context.Context, blogId int64, page, pageSize int) ([]*model.BlogComment, error)
    // SQL: WHERE blog_id=? AND parent_id=0 AND status=0 ORDER BY liked_count DESC, create_time DESC LIMIT ?,?
func (d *CommentDAO) GetRecentComments(ctx context.Context, blogId int64, page, pageSize int) ([]*model.BlogComment, error)
    // SQL: WHERE blog_id=? AND parent_id=0 AND status=0 ORDER BY create_time DESC LIMIT ?,?
func (d *CommentDAO) GetSubComments(ctx context.Context, parentId int64, page, pageSize int) ([]*model.BlogComment, error)
    // SQL: WHERE parent_id=? AND status=0 ORDER BY create_time ASC LIMIT ?,?
func (d *CommentDAO) CountSubComments(ctx context.Context, parentId int64) (int64, error)

// 评论点赞相关
func (d *CommentDAO) CreateCommentLike(ctx context.Context, userId, commentId int64) error
func (d *CommentDAO) DeleteCommentLike(ctx context.Context, userId, commentId int64) error
func (d *CommentDAO) CheckCommentLiked(ctx context.Context, userId, commentId int64) (bool, error)
func (d *CommentDAO) IncrCommentLikedCount(ctx context.Context, commentId int64) error
func (d *CommentDAO) DecrCommentLikedCount(ctx context.Context, commentId int64) error
```

### RedisDAO（在已有的 redis_dao.go 中添加）

```go
// 博客点赞（SortedSet）
func (d *RedisDAO) BlogLikeAdd(ctx context.Context, blogId, userId int64) error          // ZADD blog:like:{blogId} timestamp userId
func (d *RedisDAO) BlogLikeRemove(ctx context.Context, blogId, userId int64) error       // ZREM blog:like:{blogId} userId
func (d *RedisDAO) IsBlogLiked(ctx context.Context, blogId, userId int64) (bool, error)  // ZSCORE != nil
func (d *RedisDAO) GetBlogLikeTopN(ctx context.Context, blogId int64, n int64) ([]int64, error) // ZREVRANGE 0 n-1 → userId列表
func (d *RedisDAO) GetBlogLikeCount(ctx context.Context, blogId int64) (int64, error)    // ZCARD

// 评论点赞（Set）
func (d *RedisDAO) CommentLikeAdd(ctx context.Context, commentId, userId int64) error
func (d *RedisDAO) CommentLikeRemove(ctx context.Context, commentId, userId int64) error
func (d *RedisDAO) IsCommentLiked(ctx context.Context, commentId, userId int64) (bool, error)

// 博客详情缓存（Hash）
func (d *RedisDAO) GetBlogDetailCache(ctx context.Context, blogId int64) (*model.Blog, error)
func (d *RedisDAO) SetBlogDetailCache(ctx context.Context, blog *model.Blog, ttl time.Duration) error
func (d *RedisDAO) DelBlogDetailCache(ctx context.Context, blogId int64) error

// 评论列表缓存（List，存评论ID）
func (d *RedisDAO) GetHotCommentIds(ctx context.Context, blogId int64, start, stop int64) ([]string, error)
func (d *RedisDAO) SetHotCommentIds(ctx context.Context, blogId int64, ids []string, ttl time.Duration) error
func (d *RedisDAO) DelCommentListCache(ctx context.Context, blogId int64) error // 删hot+recent
```

---

## 五、Service 层

创建 `service/blog_service.go`，包含以下核心业务逻辑：

### 1. 点赞/取消点赞博客

```
流程：
1. ZSCORE blog:like:{blogId} userId → 判断是否已点赞（先查Redis，miss查DB）
2. 如果未点赞：
   a. DB事务内：INSERT blog_like + UPDATE blog SET liked_count=liked_count+1
   b. ZADD blog:like:{blogId} time.Now().Unix() userId
   c. 删除博客详情缓存（DelBlogDetailCache）
   d. 返回 liked=true
3. 如果已点赞（取消）：
   a. DB事务内：DELETE blog_like WHERE user_id=? AND blog_id=? + UPDATE blog SET liked_count=liked_count-1
   b. ZREM blog:like:{blogId} userId
   c. 删除博客详情缓存
   d. 返回 liked=false
```

**注意**：先DB后Redis。DB是source of truth，DB失败直接返回错误不影响Redis。

### 2. 点赞排行榜

```
流程：
1. ZREVRANGE blog:like:{blogId} 0 9 WITHSCORES → 拿到top10 userId
2. 批量查用户信息（头像+昵称，可从Redis缓存读，miss查DB）
3. 返回 [{userId, nickname, avatar, likedTime}, ...]
```

### 3. 查看博客详情

```
流程：
1. 先查Redis blog:detail:{blogId}，miss则查DB并写入Redis（TTL=30min）
2. 补充isLike字段：ZSCORE blog:like:{blogId} 当前userId → 不为nil则isLike=true
3. 补充作者信息（头像、昵称）
4. 返回完整博客详情
```

**注意**：isLike不能放进缓存，因为跟当前用户相关。

### 4. 添加评论

```
流程：
1. 参数校验（content长度1-500，parentId合法性）
2. 如果parentId!=0，校验父评论存在且status==0
3. DB事务内：
   a. INSERT blog_comment
   b. UPDATE blog SET comment_count=comment_count+1
4. 删除评论列表缓存（DelCommentListCache）
5. 返回评论id
```

### 5. 删除评论

```
流程：
1. 权限校验：评论作者本人才能删除
2. 查询该评论下子评论数量
3. DB事务内：
   a. UPDATE blog_comment SET status=2 WHERE id=?（逻辑删除）
   b. 如果是一级评论：UPDATE blog_comment SET status=2 WHERE parent_id=?
   c. UPDATE blog SET comment_count=comment_count-(1+子评论数)
4. 删除评论列表缓存 + 相关评论详情缓存
5. 返回成功
```

**注意**：逻辑删除（status=2）而非物理删除，保留回复链完整。

### 6. 点赞/取消点赞评论

```
流程与博客点赞类似，但用Set而非SortedSet：
1. SISMEMBER comment:like:{commentId} userId → 判断
2. 未点赞：事务内 INSERT comment_like + UPDATE liked_count+1，SADD
3. 已点赞：事务内 DELETE comment_like + UPDATE liked_count-1，SREM
```

### 7. 获取评论列表

```
流程：
1. 查Redis blog:comments:{blogId}:hot，miss则查DB并写入Redis（TTL=10min）
2. 拿到评论ID列表 → 批量查评论详情（comment:detail:{id}，miss查DB）
3. 对每条一级评论，补充 isLike（SISMEMBER）和子评论数量
4. 返回分页结果
```

---

## 六、Controller 层

创建 `controller/blog_controller.go`，包含以下接口：

| 方法 | 路由 | 功能 | 请求参数 |
|------|------|------|---------|
| GET | /blog/{id} | 博客详情 | path: id |
| POST | /blog/{id}/like | 点赞/取消点赞博客 | path: id, 鉴权userId |
| GET | /blog/{id}/like/top | 点赞排行榜 | path: id, query: n(默认10) |
| GET | /blog/{id}/comments/hot | 热门评论列表 | path: id, query: page, pageSize |
| GET | /blog/{id}/comments/recent | 最新评论列表 | path: id, query: page, pageSize |
| POST | /blog/{id}/comment | 添加评论 | path: id, body: {content, parentId, replyUserId} |
| DELETE | /comment/{id} | 删除评论 | path: id, 鉴权userId |
| POST | /comment/{id}/like | 点赞/取消点赞评论 | path: id, 鉴权userId |
| GET | /comment/{id}/replies | 获取子评论列表 | path: id, query: page, pageSize |

鉴权方式：从 ctx.GetInt64("userId") 获取当前登录用户ID（中间件已设置）。

---

## 七、关键设计原则（请严格遵守）

1. **先DB后Redis**：所有写操作先写MySQL，成功后再更新/删除Redis缓存。DB失败不影响Redis状态。
2. **Cache Aside模式**：读时miss查DB再写缓存，写时删除缓存让下次读重建。不要在写时更新缓存。
3. **冗余计数**：liked_count和comment_count是冗余字段，只在DB的UPDATE语句中维护，不用COUNT查询。
4. **逻辑删除**：评论和博客不做物理删除，用status字段标记。
5. **一致性保障**：Redis更新失败时，记录日志但不阻塞主流程，缓存会在下次读时重建。
6. **ID生成**：所有主键ID使用项目中已有的雪花算法生成器（utils.IDGenerator.Generate）。
7. **错误处理**：DAO层返回原始error，Service层包装业务错误信息，Controller层返回统一JSON格式。
8. **分页参数**：默认page=1, pageSize=10，pageSize上限50。

---

## 八、代码风格参考

请参考项目中已有的代码风格，特别注意：
- 函数注释使用 `// FuncName 描述` 格式
- 错误处理使用 `fmt.Errorf("描述：%v", err)` 包装
- Redis操作使用项目中已有的 RedisClient 实例
- 数据库操作使用 GORM，参考已有的 DAO 写法
- 响应格式统一：`gin.H{"code": 200, "msg": "...", "data": ...}`

请按 model → dao → service → controller 的顺序逐步实现，每完成一层确认编译通过再继续下一层。
