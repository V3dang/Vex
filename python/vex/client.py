import json
import urllib.request
import urllib.error
import urllib.parse
import uuid
import hashlib
from typing import List, Dict, Any, Optional, Union

class VexException(Exception):
    pass

class Collection:
    def __init__(self, name: str, client: "Client"):
        self.name = name
        self.client = client

    def _format_id(self, doc_id: str) -> str:
        return f"{self.name}::{doc_id}"

    def _unformat_id(self, formatted_id: str) -> str:
        if formatted_id.startswith(f"{self.name}::"):
            return formatted_id[len(self.name) + 2 :]
        return formatted_id

    def add(
        self,
        embeddings: List[List[float]],
        documents: Optional[List[str]] = None,
        metadatas: Optional[List[Dict[str, Any]]] = None,
        ids: Optional[List[str]] = None,
    ):
        if not embeddings:
            raise ValueError("embeddings cannot be empty")

        n = len(embeddings)
        if documents and len(documents) != n:
            raise ValueError("Length of documents must match embeddings")
        if metadatas and len(metadatas) != n:
            raise ValueError("Length of metadatas must match embeddings")
        if ids and len(ids) != n:
            raise ValueError("Length of ids must match embeddings")

        if not ids:
            ids = []
            for i in range(n):
                content = documents[i] if documents else str(embeddings[i])
                h = hashlib.sha256(content.encode()).hexdigest()[:12]
                ids.append(f"doc_{h}_{i}")

        for i in range(n):
            doc_id = ids[i]
            vector = embeddings[i]
            doc_text = documents[i] if documents else ""
            meta = metadatas[i] if metadatas else {}

            payload = {
                "collection": self.name,
                "id": doc_id,
                "document": doc_text,
                "metadata": meta,
            }

            self.client._request(
                "POST",
                "/vectors",
                {
                    "id": self._format_id(doc_id),
                    "vector": vector,
                    "payload": payload,
                },
            )

    def query(
        self,
        query_embeddings: Union[List[float], List[List[float]]],
        n_results: int = 10,
        where: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        if not query_embeddings:
            raise ValueError("query_embeddings cannot be empty")

        # Normalize to list of vectors
        if isinstance(query_embeddings[0], (int, float)):
            queries = [query_embeddings]
            single = True
        else:
            queries = query_embeddings
            single = False

        all_ids = []
        all_distances = []
        all_documents = []
        all_metadatas = []

        for q in queries:
            resp = self.client._request(
                "POST",
                "/vectors/search",
                {"vector": q, "k": n_results * 3},
            )

            raw_results = resp.get("results", [])
            q_ids, q_dists, q_docs, q_metas = [], [], [], []

            for r in raw_results:
                payload = r.get("payload")
                if isinstance(payload, str):
                    try:
                        payload = json.loads(payload)
                    except Exception:
                        payload = {}
                elif not isinstance(payload, dict):
                    payload = {}

                # Filter by collection namespace
                if payload.get("collection") != self.name:
                    continue

                meta = payload.get("metadata", {})
                if where:
                    match = True
                    for k, v in where.items():
                        if meta.get(k) != v:
                            match = False
                            break
                    if not match:
                        continue

                doc_id = payload.get("id", self._unformat_id(r.get("id", "")))
                q_ids.append(doc_id)
                q_dists.append(r.get("distance", 0.0))
                q_docs.append(payload.get("document", ""))
                q_metas.append(meta)

                if len(q_ids) >= n_results:
                    break

            all_ids.append(q_ids)
            all_distances.append(q_dists)
            all_documents.append(q_docs)
            all_metadatas.append(q_metas)

        if single:
            return {
                "ids": all_ids[0],
                "distances": all_distances[0],
                "documents": all_documents[0],
                "metadatas": all_metadatas[0],
            }

        return {
            "ids": all_ids,
            "distances": all_distances,
            "documents": all_documents,
            "metadatas": all_metadatas,
        }

    def get(self, id: str) -> Optional[Dict[str, Any]]:
        try:
            resp = self.client._request("GET", f"/vectors?id={urllib.parse.quote(self._format_id(id))}")
            if not resp.get("found"):
                return None
            payload = resp.get("payload", {})
            if isinstance(payload, str):
                try:
                    payload = json.loads(payload)
                except Exception:
                    payload = {}
            return {
                "id": id,
                "vector": resp.get("vector"),
                "document": payload.get("document", ""),
                "metadata": payload.get("metadata", {}),
            }
        except VexException:
            return None

    def delete(self, id: str) -> bool:
        resp = self.client._request("DELETE", f"/vectors?id={urllib.parse.quote(self._format_id(id))}")
        return resp.get("success", False)


class Client:
    def __init__(self, endpoints: Union[str, List[str]] = "http://127.0.0.1:8080"):
        if isinstance(endpoints, str):
            self.endpoints = [endpoints.rstrip("/")]
        else:
            self.endpoints = [ep.rstrip("/") for ep in endpoints]

        if not self.endpoints:
            raise ValueError("At least one endpoint must be provided")

        self.current_idx = 0
        self._refresh_cluster_nodes()

    def _refresh_cluster_nodes(self):
        try:
            info = self.get_cluster_info()
            nodes = info.get("nodes", [])
            for n in nodes:
                http_addr = n.get("http_addr")
                if http_addr:
                    url = f"http://{http_addr}"
                    if url not in self.endpoints:
                        self.endpoints.append(url)
        except Exception:
            pass

    def _request(self, method: str, path: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        last_error = None
        attempts = len(self.endpoints)

        for _ in range(attempts):
            endpoint = self.endpoints[self.current_idx % len(self.endpoints)]
            url = f"{endpoint}{path}"
            headers = {"Content-Type": "application/json"}
            body = json.dumps(data).encode("utf-8") if data else None

            req = urllib.request.Request(url, data=body, headers=headers, method=method)

            try:
                with urllib.request.urlopen(req, timeout=5) as response:
                    res_body = response.read().decode("utf-8")
                    return json.loads(res_body) if res_body else {}
            except urllib.error.HTTPError as e:
                err_text = e.read().decode("utf-8")
                try:
                    err_json = json.loads(err_text)
                    raise VexException(err_json.get("error", err_text))
                except json.JSONDecodeError:
                    raise VexException(f"HTTP {e.code}: {err_text}")
            except (urllib.error.URLError, ConnectionError, TimeoutError) as e:
                last_error = e
                # Fail over to next peer endpoint
                self.current_idx = (self.current_idx + 1) % len(self.endpoints)

        raise VexException(f"Failed to connect to any cluster endpoint: {last_error}")

    def get_or_create_collection(self, name: str) -> Collection:
        return Collection(name=name, client=self)

    def get_cluster_info(self) -> Dict[str, Any]:
        return self._request("GET", "/cluster")

    def health(self) -> Dict[str, Any]:
        return self._request("GET", "/health")
