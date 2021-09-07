## tke-auth-controller
텐센트 클러스터에서 SubAccountId 가 담겨있는 config 을 받아, ClusteRoleBinding 으로 변환하는 컨트롤러 입니다.

## 사용 방법

아래 내용을 담은 configMap 을 클러스터에 배포하면, controller 가 해당 object 를 인식하여 CRB 로 변환시켜 줍니다.  
반드시 annotations.tke-auth/binding 이 있어야 인식합니다.  
아무 namespace 에 configMap 을 배포하여도 무방합니다.

`configMap-sample.yaml` 을 참고하여 작성하셔도 좋습니다.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap-sample
  annotations:
    tke-auth/binding: "true" # 값은 무엇을 적어도 상관 없고, 해당 key 가 존재하면 됩니다.
data:
  # 생성/업데이트 될 clusterRoleBinding 의 이름입니다. 중복되는 이름이 없도록 주의해주세요.
  bindingName: "xtrm-platform-team-default"
  # binding 될 clusterRole 의 이름입니다.
  roleName: "xtrm:user:full-control"
  # 사용자 목록입니다. subAccountId 를 넣거나, userId (보통 email 임) 을 넣을 수 있습니다.
  users: |
    defaultUserValueType: subAccountId
    users:
      - valueType: subAccountId
        value: "200020745365"
      - value: "384273412432" # valueType 를 지정하지 않을 경ㅇ, defaultUservalueType 를 이용합니다. 
      - valueType: email
        value: "foobar@pubg.com"
```
