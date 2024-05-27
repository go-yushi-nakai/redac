package redac

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	RedashJobStatusPending   = 1
	RedashJobStatusStarted   = 2
	RedashJobStatusSuccess   = 3
	RedashJobStatusFailure   = 4
	RedashJobStatusCancelled = 5
)

type RedashPostQueryResultRequest struct {
	ApplyAutoLimit bool           `json:"apply_auto_limit"`
	DataSourceID   int            `json:"data_source_id"`
	MaxAge         int            `json:"max_age"`
	Parameters     map[string]any `json:"parameters"`
	Query          string         `json:"query"`
}

type RedashGetJobResponse struct {
	Job struct {
		ID            string `json:"id"`
		UpdatedAt     any    `json:"updated_at"`
		Status        int    `json:"status"`
		Error         string `json:"error"`
		Result        int    `json:"result"`
		QueryResultID int    `json:"query_result_id"`
	} `json:"job"`
}

type RedashGetQueryResultResponse struct {
	QueryResult struct {
		ID        int    `json:"id"`
		QueryHash string `json:"query_hash"`
		Query     string `json:"query"`
		Data      struct {
			Columns []struct {
				Name         string `json:"name"`
				FriendlyName string `json:"friendly_name"`
				Type         string `json:"type"`
			} `json:"columns"`
			Rows []map[string]any `json:"rows"`
		} `json:"data"`
	} `json:"query_result"`
}

func (r *RedashGetQueryResultResponse) GetTable() [][]string {
	data := r.QueryResult.Data

	table := make([][]string, len(data.Rows)+1)
	tableHead := make([]string, len(data.Columns))
	col2type := make(map[string]string, len(data.Rows))
	for i, col := range data.Columns {
		tableHead[i] = col.FriendlyName
		col2type[col.Name] = col.Type
	}

	table[0] = tableHead
	for i, row := range data.Rows {
		tableRow := make([]string, len(data.Columns))
		for j, column := range data.Columns {
			v := row[column.Name]
			if v == nil {
				tableRow[j] = ""
				continue
			}
			switch col2type[column.Name] {
			case "integer":
				tableRow[j] = fmt.Sprint(int(v.(float64)))
			default:
				tableRow[j] = fmt.Sprint(v)
			}
		}
		table[i+1] = tableRow
	}
	return table
}

type RedashClient struct {
	Endpoint string
	APIKey   string
	Logger   *slog.Logger
}

func NewRedashClient(endpoint, apiKey string, logger *slog.Logger) (*RedashClient, error) {
	rc := &RedashClient{}
	rc.Endpoint = endpoint
	rc.APIKey = apiKey
	rc.Logger = logger
	return rc, nil
}

func (rc *RedashClient) GetDataSources(ctx context.Context) ([]map[string]any, error) {
	resp, err := rc.doRequest(ctx, http.MethodGet, "data_sources", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources. %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get data source, status=%d", resp.StatusCode)
	}
	var data []map[string]any
	if err := rc.unmarshalResponse(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response. %w", err)
	}
	return data, nil
}

func (rc *RedashClient) QueryAndWaitResult(ctx context.Context, req RedashPostQueryResultRequest) (*RedashGetQueryResultResponse, error) {
	rc.Logger.Debug("QueryAndWaitResult", "req", req)
	job, err := rc.PostQueryResults(ctx, req)
	if err != nil {
		return nil, err
	}
	jobID := job.Job.ID

	for {
		time.Sleep(time.Second)
		job, err = rc.GetJob(ctx, jobID)

		select {
		case <-ctx.Done():
			rc.cleanupJob(jobID)
		default:
		}

		if err != nil {
			return nil, err
		}

		if job.Job.Status == RedashJobStatusSuccess {
			break
		}

		if job.Job.Status == RedashJobStatusFailure {
			return nil, fmt.Errorf("job is failed: %s", job.Job.Error)
		}
	}
	return rc.GetQueryResult(ctx, job.Job.QueryResultID)
}

func (rc *RedashClient) cleanupJob(jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rc.DeleteJob(ctx, jobID); err != nil {
		rc.Logger.Error("failed to delete job", "job_id", jobID, "err", err)
	}
	rc.Logger.Warn("job is cancelled", "job_id", jobID)
}

func (rc *RedashClient) PostQueryResults(ctx context.Context, req RedashPostQueryResultRequest) (*RedashGetJobResponse, error) {
	resp, err := rc.doRequest(ctx, http.MethodPost, "query_results", req)
	if err != nil {
		return nil, fmt.Errorf("failed to post query results. %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("failed to post query result, status=%d, body=%s", resp.StatusCode, b)
	}
	var data RedashGetJobResponse
	if err := rc.unmarshalResponse(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response. %w", err)
	}
	if data.Job.Status == RedashJobStatusFailure {
		return nil, fmt.Errorf("job failed: %s", data.Job.Error)
	}
	return &data, nil
}

func (rc *RedashClient) GetJob(ctx context.Context, id string) (*RedashGetJobResponse, error) {
	api := fmt.Sprintf("jobs/%s", id)
	resp, err := rc.doRequest(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %s: %s", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("failed to get job, status=%d, body=%s", resp.StatusCode, b)
	}
	var data RedashGetJobResponse
	if err := rc.unmarshalResponse(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response. %w", err)
	}
	return &data, nil
}

func (rc *RedashClient) DeleteJob(ctx context.Context, id string) error {
	api := fmt.Sprintf("jobs/%s", id)
	resp, err := rc.doRequest(ctx, http.MethodDelete, api, nil)
	if err != nil {
		return fmt.Errorf("failed to delete job %s: %s", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("failed to delete job, status=%d, body=%s", resp.StatusCode, b)
	}
	return nil
}

func (rc *RedashClient) GetQueryResult(ctx context.Context, id int) (*RedashGetQueryResultResponse, error) {
	api := fmt.Sprintf("query_results/%d", id)
	resp, err := rc.doRequest(ctx, http.MethodGet, api, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get query result at request, id=%d: %w", id, err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("failed to get query result, status=%d, body=%s", resp.StatusCode, b)
	}
	var data RedashGetQueryResultResponse
	if err := rc.unmarshalResponse(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response. %w", err)
	}
	return &data, nil
}

func (rc *RedashClient) doRequest(ctx context.Context, method, api string, reqData any) (*http.Response, error) {
	b, _ := json.Marshal(reqData)
	br := bytes.NewBuffer(b)
	req, err := rc.newRequest(method, api, br)
	if err != nil {
		return nil, fmt.Errorf("failed to create request. %s %s, %w", req.Method, req.URL, err)
	}
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request. %s %s, %w", req.Method, req.URL, err)
	}
	return resp, nil
}

func (rc *RedashClient) unmarshalResponse(resp *http.Response, v any) error {
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	rc.Logger.Debug("unmarshal response", "body", string(b))
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("failed to unmarshal response body, err=%w, body=%s", err, string(b))
	}
	return nil

}
func (rc *RedashClient) newRequest(method, api string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s/api/%s", rc.Endpoint, api)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request. %s %s, %w", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Key %s", rc.APIKey))
	return req, nil
}
