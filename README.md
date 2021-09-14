## tke-auth-controller
텐센트 클러스터에서 SubAccountId 가 담겨있는 config 을 받아, ClusteRoleBinding 으로 변환하는 컨트롤러 입니다.

## 사용 방법

아래 내용을 담은 configMap 을 클러스터에 배포하면, controller 가 해당 object 를 인식하여 CRB 로 변환시켜 줍니다.  
반드시 annotations.tke-auth/binding 이 있어야 인식합니다.  
아무 namespace 에 configMap 을 배포하여도 무방합니다.
