# LeetcodeDashboard



## DataSource:

### ProblemInfo:

Request:

```
https://leetcode.com/api/problems/all/
```


### Submissions:

https://leetcode.com/submissions/#/1

Request(with cookie):

https://leetcode.com/api/submissions/?offset=0&limit=20&lastkey=




### TermUI library modification:

For barchart.go and stacked_barchart.go:

Add the following in Draw()
```go
	if maxVal == 0 {
		maxVal = 1
	}
```

https://github.com/gizak/termui/issues/245