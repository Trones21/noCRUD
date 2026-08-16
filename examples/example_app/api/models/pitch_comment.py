from django.db import models
from api.models.backstory_pitch import BackstoryPitch
from api.models.users import Users
from rest_framework import serializers
from rest_framework.viewsets import ModelViewSet
from rest_framework.permissions import IsAuthenticated


class PitchComment(models.Model):
    pitch = models.ForeignKey(
        BackstoryPitch, on_delete=models.CASCADE, related_name="comments"
    )
    user = models.ForeignKey(Users, on_delete=models.CASCADE)
    text = models.TextField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "pitch_comment"


class PitchCommentSerializer(serializers.ModelSerializer):
    class Meta:
        model = PitchComment
        fields = "__all__"


class PitchCommentViewSet(ModelViewSet):
    queryset = PitchComment.objects.all()
    serializer_class = PitchCommentSerializer
    permission_classes = [IsAuthenticated]

    def perform_create(self, serializer):
        # Save comment with user injected
        comment = serializer.save(user=self.request.user)

        # Lock pitch if this is the first interaction
        comment.pitch.lock_if_interacted()
