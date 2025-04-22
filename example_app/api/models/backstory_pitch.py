from django.db import models
from django.core.exceptions import ValidationError
from rest_framework import serializers, viewsets
from .character import Character
from django.contrib.auth import get_user_model

User = get_user_model()


class BackstoryPitch(models.Model):
    character = models.ForeignKey(
        Character, on_delete=models.CASCADE, related_name="pitches"
    )
    user = models.ForeignKey(User, on_delete=models.CASCADE)
    pitch_text = models.TextField()

    upvotes = models.IntegerField(default=0)
    created_at = models.DateTimeField(auto_now_add=True)
    edited_at = models.DateTimeField(auto_now=True)

    locked = models.BooleanField(default=False)

    def save(self, *args, **kwargs):
        if self.pk:
            original = BackstoryPitch.objects.get(pk=self.pk)
            if original.locked:
                raise ValidationError("This pitch is locked and cannot be edited.")
        super().save(*args, **kwargs)

    def __str__(self):
        return f"Pitch for {self.character.name} by {self.user.username}"

    def lock_if_interacted(self):
        if not self.locked and (self.upvotes > 0 or self.comments.exists()):  # type: ignore
            self.locked = True
            self.save()

    class Meta:
        managed = True
        db_table = "pitch"
        ordering = ["-upvotes", "-created_at"]


class BackstoryPitchSerializer(serializers.ModelSerializer):
    class Meta:
        model = BackstoryPitch
        fields = "__all__"


class BackstoryPitchViewSet(viewsets.ModelViewSet):
    queryset = BackstoryPitch.objects.all()
    serializer_class = BackstoryPitchSerializer
